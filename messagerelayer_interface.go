package messagerelayer

type NetworkSocket interface {
	Read() (Message, error)
}

type MessageType int

type Message struct {
	Type MessageType
	Data []byte
}

type MessageRelayer interface {
	SubscribeToMessages(msgType MessageType, messages chan<- Message)
}
