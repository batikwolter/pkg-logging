package observe

const (
	EventRPCServerCompleted = "rpc.server.completed"
	EventRPCClientCompleted = "rpc.client.completed"
)

func Event(name string) Field {
	return String("event", name)
}
