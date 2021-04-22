package rabbitmq

func UseDefaultTopology(exchange string) func(*Transport) {
	return func(rmq *Transport) {
		rmq.topology = &DefaultTopology{
			Exchange:  exchange,
			Transport: rmq,
		}
	}
}

func UsePriorityQueue(maxPriority uint8) func(*Transport) {
	return func(rmq *Transport) {
		if rmq.InputQueue.Args == nil {
			rmq.InputQueue.Args = make(map[string]interface{})
		}
		rmq.InputQueue.Args["x-max-priority"] = maxPriority
	}
}
