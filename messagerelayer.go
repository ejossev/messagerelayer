package messagerelayer

import "sync"

type SizedMessageQueue struct {
	messages []Message
	size     int
	first    int
	maxSize  int
}

func NewMessageQueue(maxSize int) *SizedMessageQueue {
	rv := new(SizedMessageQueue)
	rv.maxSize = maxSize
	rv.size = 0
	rv.first = 0
	rv.messages = make([]Message, maxSize)
	return rv
}

func (q *SizedMessageQueue) AddMessage(m Message) {
	nextPosition := (q.first + q.size) % q.maxSize
	if q.size == q.maxSize {
		q.first = (q.first + 1) % q.maxSize
	} else {
		q.size = q.size + 1
	}
	q.messages[nextPosition] = m
}

func (q *SizedMessageQueue) GetMessages() []Message {
	rv := make([]Message, q.size)
	for i := 0; i < q.size; i++ {
		pos := (q.first + i) % q.maxSize
		rv[i] = q.messages[pos]
	}
	return rv
}

func (q *SizedMessageQueue) ClearQueue() {
	q.size = 0
	q.first = 0
}

type MessageSubscriber struct {
	active   bool
	messages chan<- Message
}

type MyMessageRelayer struct {
	socket         NetworkSocket
	subscribers    map[MessageType][]MessageSubscriber
	subscriberLock sync.Mutex
}

const (
	StartNewRound MessageType = 1 << iota
	ReceivedAnswer
)

func NewMessageRelayer(socket NetworkSocket) *MyMessageRelayer {
	rv := new(MyMessageRelayer)
	rv.socket = socket
	rv.subscribers = make(map[MessageType][]MessageSubscriber)
	rv.subscribers[StartNewRound] = make([]MessageSubscriber, 0, 10)
	rv.subscribers[ReceivedAnswer] = make([]MessageSubscriber, 0, 10)
	return rv
}

func (mr *MyMessageRelayer) SubscribeToMessages(msgType MessageType, messages chan<- Message) {
	sub := MessageSubscriber{true, messages}
	mr.subscriberLock.Lock()
	mr.subscribers[msgType] = append(mr.subscribers[msgType], sub)
	mr.subscriberLock.Unlock()
}

func (mr *MyMessageRelayer) process() {
	snrQueue := NewMessageQueue(2)
	raQueue := NewMessageQueue(1)

	go func() {
		for {
			snrQueue.ClearQueue()
			raQueue.ClearQueue()
			m, err := mr.socket.Read()
			for err == nil {
				switch m.Type {
				case StartNewRound:
					snrQueue.AddMessage(m)
				case ReceivedAnswer:
					raQueue.AddMessage(m)
				}
				m, err = mr.socket.Read()
			}
			mr.subscriberLock.Lock()
			messagesToSend := append(snrQueue.GetMessages(), raQueue.GetMessages()...)
			for _, m := range messagesToSend {
				for i := 0; i < len(mr.subscribers[m.Type]); i++ {
					subscriber := &mr.subscribers[m.Type][i]
					if subscriber.active {
						select {
						case subscriber.messages <- m:
						default:
							subscriber.active = false
							close(subscriber.messages)
						}
					}
				}
			}
			mr.subscriberLock.Unlock()
		}
	}()
}
