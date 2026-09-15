package rabbitadapter

import (
	"context"
	"encoding/json"
	"log"

	"erp/pkg/rabbit"
	"erp/services/stock-service/internal/application"
	"erp/services/stock-service/internal/domain"
)

type Publisher struct {
	client *rabbit.Client
}

func NewPublisher(client *rabbit.Client) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishReserved(ctx context.Context, ev domain.OrderEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.client.Publish(ctx, "stock.exchange", "stock.reserved", body)
}

func Consume(client *rabbit.Client, svc *application.Service) error {
	if err := client.EnsureTopology("sales.exchange", "stock.order-created.queue", "sales.order.created"); err != nil {
		return err
	}
	if err := client.EnsureTopology("sales.exchange", "stock.order-created.queue", "sales.order.updated"); err != nil {
		return err
	}
	if err := client.EnsureTopology("sales.exchange", "stock.order-created.queue", "sales.order.cancelled"); err != nil {
		return err
	}
	if err := client.Consume("stock.order-created.queue", func(body []byte) error {
		var ev domain.OrderEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return err
		}
		if err := svc.ReserveOrder(context.Background(), ev); err != nil {
			log.Printf("reserve order %s: %v", ev.OrderID, err)
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if err := client.EnsureTopology("invoicing.exchange", "stock.nfe-issued.queue", "invoicing.nfe.issued"); err != nil {
		return err
	}
	return client.Consume("stock.nfe-issued.queue", func(body []byte) error {
		var ev domain.InvoiceEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return err
		}
		if err := svc.ConfirmInvoice(context.Background(), ev); err != nil {
			log.Printf("confirm invoice %s: %v", ev.InvoiceID, err)
			return err
		}
		return nil
	})
}
