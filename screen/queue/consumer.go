package queue

import "github.com/streadway/amqp"

// RegisterRabbitMQConsumer 注册消费者
func RegisterRabbitMQConsumer() {

	//字节充值、转款、退款截图
	zjRabbit := NewRabbitMQ("yoyo_zj_exchange", "yoyo_zj1_route", "yoyo_zj1_queue")
	zjRabbit.Consume(func(d amqp.Delivery) {
		//字节充值、转款、退款截图
		SnapshotZjConsumer(d)
	})

	//快手充值、转款、退款截图
	ksRabbit := NewRabbitMQ("yoyo_ks_exchange", "yoyo_ks_route", "yoyo_ks_queue")
	ksRabbit.Consume(func(d amqp.Delivery) {
		SnapshotKSConsumer(d)
	})
}
