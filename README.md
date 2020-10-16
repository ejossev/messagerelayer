# MessageRelayer

Usage:
```
mr := NewMessageRelayer(networkSocket)
mr.process()
```

The clients subscribe with 
```
messages := make(chan Message, 3)
mr.SubscribeToMessages(StartNewRound, messages)
mr.SubscribeToMessages(ReceivedAnswer, messages)
m := <-messages
...
```


