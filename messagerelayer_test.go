package messagerelayer

import "testing"
import "time"
import "fmt"
import "errors"
import "github.com/stretchr/testify/assert"

//import "github.com/stretchr/testify/require"

func TestQueue_Size1(t *testing.T) {
	queue := NewMessageQueue(1)
	m1 := Message{1, []byte{0}}
	m2 := Message{1, []byte{1}}

	assert.Equal(t, queue.size, 0)
	queue.AddMessage(m1)
	assert.Equal(t, queue.size, 1)
	queue.AddMessage(m2)
	assert.Equal(t, queue.size, 1)
	msgs := queue.GetMessages()
	assert.Equal(t, len(msgs), 1)
	assert.Equal(t, msgs[0], m2)

	queue.ClearQueue()
	assert.Equal(t, queue.size, 0)
	msgs = queue.GetMessages()
	assert.Equal(t, len(msgs), 0)
}

func TestQueue_Size2(t *testing.T) {
	queue := NewMessageQueue(2)
	m1 := Message{1, []byte{0}}
	m2 := Message{1, []byte{1}}
	m3 := Message{1, []byte{2}}

	assert.Equal(t, queue.size, 0)
	queue.AddMessage(m1)
	assert.Equal(t, queue.size, 1)
	queue.AddMessage(m2)
	assert.Equal(t, queue.size, 2)
	msgs := queue.GetMessages()
	assert.Equal(t, len(msgs), 2)
	assert.Equal(t, msgs[0], m1)
	assert.Equal(t, msgs[1], m2)

	queue.ClearQueue()
	assert.Equal(t, queue.size, 0)
	queue.AddMessage(m1)
	queue.AddMessage(m2)
	queue.AddMessage(m3)
	assert.Equal(t, queue.size, 2)
	msgs = queue.GetMessages()
	assert.Equal(t, msgs[0], m2)
	assert.Equal(t, msgs[1], m3)

}

type TestNetworkSocket struct {
	messages []Message
	iterator int
}

func (tns *TestNetworkSocket) Read() (Message, error) {
	if tns.iterator == len(tns.messages) {
		return Message{-1, []byte{}}, errors.New("Out of messages")
	}
	tns.iterator = tns.iterator + 1
	return tns.messages[tns.iterator-1], nil
}

func TestMessageRelayer_start(t *testing.T) {
	tns := new(TestNetworkSocket)
	tns.messages = []Message{{1, []byte{}}, {2, []byte{}}, {1, []byte{}}, {1, []byte{}}, {2, []byte{}}}
	tns.iterator = 0

	mr := NewMessageRelayer(tns)
	fmt.Printf("Preparing clients\n")

	sync_channel_client1 := make(chan bool)
	sync_channel_client2 := make(chan bool)
	sync_channel_client3 := make(chan bool)
	sync_channel_client4 := make(chan bool)

	// Mock client 1 - subscribed to both message types
	go func(sync_channel chan bool, mr MessageRelayer) {
		messages := make(chan Message, 3) //Be ready to recieve all three messages at once
		mr.SubscribeToMessages(StartNewRound, messages)
		mr.SubscribeToMessages(ReceivedAnswer, messages)
		fmt.Printf("About to receive messages in client 1\n")
		m1, m2, m3 := <-messages, <-messages, <-messages
		assert.Equal(t, m1.Type, StartNewRound)
		assert.Equal(t, m2.Type, StartNewRound)
		assert.Equal(t, m3.Type, ReceivedAnswer)
		fmt.Printf("Client1 received successfully\n")
		sync_channel <- true
		_ = <-sync_channel //Wait for next round
		fmt.Printf("About to receive messages in client 1\n")
		m1, m2, m3 = <-messages, <-messages, <-messages
		assert.Equal(t, m1.Type, StartNewRound)
		assert.Equal(t, m2.Type, StartNewRound)
		assert.Equal(t, m3.Type, ReceivedAnswer)
		fmt.Printf("Client1 received successfully\n")
		sync_channel <- true
	}(sync_channel_client1, mr)

	// Mock client 2 - subscribed to StartNewRound only
	go func(sync_channel chan bool, mr MessageRelayer) {
		messages := make(chan Message, 2) //Be ready to recieve all three messages at once
		mr.SubscribeToMessages(StartNewRound, messages)
		fmt.Printf("About to receive messages in client 2\n")
		m1, m2 := <-messages, <-messages
		assert.Equal(t, m1.Type, StartNewRound)
		assert.Equal(t, m2.Type, StartNewRound)
		fmt.Printf("Client2 received successfully\n")
		sync_channel <- true
		_ = <-sync_channel //Wait for next round
		fmt.Printf("About to receive messages in client 2\n")
		m1, m2 = <-messages, <-messages
		assert.Equal(t, m1.Type, StartNewRound)
		assert.Equal(t, m2.Type, StartNewRound)
		fmt.Printf("Client2 received successfully\n")
		sync_channel <- true
	}(sync_channel_client2, mr)

	// Mock client 3 - subscribed to ReceivedAnswer
	go func(sync_channel chan bool, mr MessageRelayer) {
		messages := make(chan Message) //Be ready to recieve all three messages at once
		mr.SubscribeToMessages(ReceivedAnswer, messages)
		fmt.Printf("About to receive messages in client 3\n")
		m1 := <-messages
		assert.Equal(t, m1.Type, ReceivedAnswer)
		fmt.Printf("Client3 received successfully\n")
		sync_channel <- true
		_ = <-sync_channel //Wait for next round
		fmt.Printf("About to receive messages in client 3\n")
		m1 = <-messages
		assert.Equal(t, m1.Type, ReceivedAnswer)
		fmt.Printf("Client3 received successfully\n")
		sync_channel <- true
	}(sync_channel_client3, mr)

	// Mock client 4 - subscribe but do not listen
	go func(sync_channel chan bool, mr MessageRelayer) {
		messages := make(chan Message)
		mr.SubscribeToMessages(StartNewRound, messages)
		// Not receiving message here
		sync_channel <- true
		_ = <-sync_channel //Wait for next round
		fmt.Printf("About to receive messages in client 4\n")
		m1 := <-messages
		assert.Equal(t, m1.Type, MessageType(0)) //On closed channel we receive 0 value
		sync_channel <- true
	}(sync_channel_client4, mr)

	time.Sleep(time.Second / 10)
	mr.process()
	time.Sleep(time.Second / 10)
	fmt.Printf("Waiting for clients\n")
	_ = <-sync_channel_client1
	fmt.Printf("   client 1 done\n")
	_ = <-sync_channel_client2
	fmt.Printf("   client 2 done\n")
	_ = <-sync_channel_client3
	fmt.Printf("   client 3 done\n")
	_ = <-sync_channel_client4
	fmt.Printf("   client 4 done\n")

	assert.Equal(t, tns.iterator, 5)
	sync_channel_client1 <- true
	sync_channel_client2 <- true
	sync_channel_client3 <- true
	sync_channel_client4 <- true
	// Reset the test network socker
	tns.iterator = 0

	fmt.Printf("Waiting for clients\n")
	_ = <-sync_channel_client1
	fmt.Printf("   client 1 done\n")
	_ = <-sync_channel_client2
	fmt.Printf("   client 2 done\n")
	_ = <-sync_channel_client3
	fmt.Printf("   client 3 done\n")
	_ = <-sync_channel_client4
	fmt.Printf("   client 4 done\n")

}
