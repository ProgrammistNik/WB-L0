package db

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"l0/internal/config"
	"l0/internal/interfaces"
	"l0/internal/models"

	_ "database/sql"

	_ "github.com/jackc/pgx/v5/pgxpool"
)

// An OrderRepo is a repository pattern implementation for working with database
type OrderRepo struct {
	db *DB
}

// NewOrderRepo creates a new instance of OrderRepo with specified configuration
func NewOrderRepo(ctx context.Context, cfg *config.Config) (*OrderRepo, error) {
	db, err := NewDBWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &OrderRepo{db}, nil
}

// SaveOrder adds an order to the database using transaction
func (o *OrderRepo) SaveOrder(ctx context.Context, order *models.Order) error {
	_, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			dID, err := o.insertDelivery(ctx, tx, &order.Delivery)
			if err != nil {
				return nil, err
			}

			err = o.insertOrder(ctx, tx, order, dID)
			if err != nil {
				return nil, err
			}

			err = o.insertItems(ctx, tx, order.Items)
			if err != nil {
				return nil, err
			}

			err = o.insertPayment(ctx, tx, &order.Payment)
			if err != nil {
				return nil, err
			}

			return nil, err
		},
	)
	return err
}

// insertOrder is a private method to add order to the database with payment, items and delivery already inserted
func (o *OrderRepo) insertOrder(
	ctx context.Context, q interfaces.Queryable, order *models.Order, deliveryID int64,
) error {
	query := `
		INSERT INTO orders (order_uid, track_number, entry, delivery_id, locale, internal_signature, customer_id, 
			delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (order_uid) DO NOTHING;
	`

	_, err := q.Exec(
		ctx, query, order.OrderUID, order.TrackNumber, order.Entry, deliveryID, order.Locale,
		order.InternalSignature, order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID,
		order.DateCreated, order.OofShard,
	)
	return err
}

// InsertPayment is a public method to insert payment into the database using transaction
func (o *OrderRepo) InsertPayment(ctx context.Context, payment *models.Payment) error {
	_, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			return nil, o.insertPayment(ctx, tx, payment)
		},
	)
	return err
}

// insertPayment is a private method to insert payment into the database with specified querier
func (o *OrderRepo) insertPayment(ctx context.Context, q interfaces.Queryable, payment *models.Payment) error {
	query := `
		INSERT INTO payments (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, 
			goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (transaction) DO NOTHING;
	`

	_, err := q.Exec(
		ctx, query, payment.Transaction, payment.RequestID, payment.Currency, payment.Provider,
		payment.Amount, payment.PaymentDt, payment.Bank, payment.DeliveryCost, payment.GoodsTotal, payment.CustomFee,
	)
	return err
}

// InsertItems is a public method to insert list of items into the database using transaction
func (o *OrderRepo) InsertItems(ctx context.Context, items []models.Item) error {
	_, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			return nil, o.insertItems(ctx, tx, items)
		},
	)
	return err
}

// insertItems is a private method to insert list of items into the database with specified querier
func (o *OrderRepo) insertItems(ctx context.Context, q interfaces.Queryable, items []models.Item) error {
	_, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			for idx := range items {
				err := o.insertItem(ctx, q, &items[idx])
				if err != nil {
					return nil, err
				}

			}
			return nil, nil
		},
	)
	return err
}

// InsertItem is a public method to insert item into the database using transaction
func (o *OrderRepo) InsertItem(ctx context.Context, item *models.Item) error {
	_, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			return nil, o.insertItem(ctx, tx, item)
		},
	)
	return err
}

// insertItem is a private method to insert item into the database with specified querier
func (o *OrderRepo) insertItem(ctx context.Context, q interfaces.Queryable, item *models.Item) error {
	query := `
		INSERT INTO items (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (chrt_id) DO NOTHING;
	`

	_, err := q.Exec(
		ctx, query, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale, item.Size,
		item.TotalPrice, item.NmID, item.Brand, item.Status,
	)
	return err
}

// InsertDelivery is a public method to insert delivery into the database using transaction
func (o *OrderRepo) InsertDelivery(ctx context.Context, q interfaces.Queryable, delivery *models.Delivery) error {
	_, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			_, err := o.insertDelivery(ctx, tx, delivery)
			return nil, err
		},
	)
	return err
}

// insertDelivery is a private method to insert delivery into the database with specified querier
func (o *OrderRepo) insertDelivery(ctx context.Context, q interfaces.Queryable, delivery *models.Delivery) (
	int64, error,
) {
	query := `
		INSERT INTO deliveries (name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
		RETURNING id
	`
	var id int64 = 0
	err := q.QueryRow(
		ctx, query, delivery.Name, delivery.Phone, delivery.Zip, delivery.City, delivery.Address,
		delivery.Region, delivery.Email,
	).Scan(&id)
	return id, err
}

// GetOrder returns order by orderUID from the database using transaction
func (o *OrderRepo) GetOrder(ctx context.Context, orderUid string) (*models.Order, error) {
	query := `
		SELECT
			o.*,
			d.*,
			p.*,
			jsonb_agg(jsonb_build_object(
				'chrt_id', i.chrt_id,
				'track_number', i.track_number,
				'price', i.price,
				'rid', i.rid,
				'name', i.name,
				'sale', i.sale,
				'size', i.size,
				'total_price', i.total_price,
				'nm_id', i.nm_id,
				'brand', i.brand,
				'status', i.status
			)) AS items	
		FROM orders o
		LEFT JOIN deliveries d ON o.delivery_id = d.id
		LEFT JOIN payments p ON o.order_uid = p.transaction
		LEFT JOIN items i ON i.track_number = o.track_number
		WHERE o.order_uid=$1
		GROUP BY o.order_uid, d.id, p.transaction
	`

	order, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var order models.Order
			err := pgxscan.Select(ctx, tx, &order, query)
			if err != nil {
				return nil, err
			}
			return &order, nil
		},
	)

	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, pgx.ErrNoRows
	}

	return order.(*models.Order), err
}

// GetDelivery returns delivery by id from the database using transaction
func (o *OrderRepo) GetDelivery(ctx context.Context, id string) (*models.Delivery, error) {
	query := `
		SELECT
			d.*
		FROM deliveries d
		WHERE d.id=$1
	`

	delivery, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var delivery models.Delivery

			err := pgxscan.Select(ctx, tx, &delivery, query)
			if err != nil {
				return nil, err
			}
			return &delivery, nil
		},
	)

	if err != nil {
		return nil, err
	}
	if delivery == nil {
		return nil, pgx.ErrNoRows
	}

	return delivery.(*models.Delivery), err
}

// GetItems returns list of items by list of chrtIDs from the database using transaction
func (o *OrderRepo) GetItems(ctx context.Context, chrtID []int64) ([]models.Item, error) {
	query := `
		SELECT
			d.*
		FROM deliveries d
		WHERE d.id=ANY($1)
	`

	items, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var items []models.Item
			err := pgxscan.Select(ctx, tx, items, query)
			if err != nil {
				return nil, err
			}
			return items, nil
		},
	)

	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, pgx.ErrNoRows
	}

	return items.([]models.Item), err
}

// GetItem returns item by chrtID from the database using transaction
func (o *OrderRepo) GetItem(ctx context.Context, chrtID int64) (*models.Item, error) {
	query := `
		SELECT
			i.*
		FROM items i
		WHERE i.chrt_id=$1
	`

	item, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var item models.Item
			err := pgxscan.Select(ctx, tx, &item, query)
			if err != nil {
				return nil, err
			}
			return &item, nil
		},
	)

	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, pgx.ErrNoRows
	}

	return item.(*models.Item), err
}

// GetPayment returns payment by transaction from the database using transaction
func (o *OrderRepo) GetPayment(ctx context.Context, transaction string) (*models.Payment, error) {
	query := `
		SELECT
			p.*
		FROM payments p
		WHERE p.transaction=$1
	`

	payment, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var payment models.Payment
			err := pgxscan.Select(ctx, tx, &payment, query, transaction)
			if err != nil {
				return nil, err
			}
			return &payment, nil
		},
	)

	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, pgx.ErrNoRows
	}

	return payment.(*models.Payment), err
}

// GetNOrders returns list of n orders from the database using transaction
func (o *OrderRepo) GetNOrders(ctx context.Context, n int) ([]models.Order, error) {
	query := `
		SELECT *
		FROM orders
		LIMIT $1
	`

	orders, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var orders []models.Order
			err := pgxscan.Select(ctx, tx, &orders, query, n)
			if err != nil {
				return nil, err
			}
			return orders, err
		},
	)
	if err != nil {
		return []models.Order{}, err
	}
	if orders == nil {
		return []models.Order{}, nil
	}
	return orders.([]models.Order), err
}

// GetAllOrders returns list of all orders from the database using transaction
func (o *OrderRepo) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	query := `
		SELECT *
		FROM orders
	`

	orders, err := o.db.WithTx(
		ctx, func(tx pgx.Tx) (any, error) {
			var orders []models.Order
			err := pgxscan.Select(ctx, tx, &orders, query)
			if err != nil {
				return nil, err
			}
			return orders, err
		},
	)

	if err != nil {
		return []models.Order{}, err
	}
	if orders == nil {
		return []models.Order{}, nil
	}
	return orders.([]models.Order), err
}