package inventory

import "net/http"

const (
	PathInventory  = "/api/inventory"
	RouteList      = http.MethodGet + " " + PathInventory
	QuerySearch    = "search"
	QuerySort      = "sort"
	QueryDirection = "direction"
)

const (
	ColumnName     Column = "name"
	ColumnQuantity Column = "quantity"
	ColumnStatus   Column = "status"
)

const (
	DirectionAscending  Direction = "ascending"
	DirectionDescending Direction = "descending"
)

const (
	StatusInStock    Status = "In stock"
	StatusLow        Status = "Low"
	StatusOutOfStock Status = "Out of stock"
)

const (
	CodeInvalidQuery    = "invalid_query"
	MsgInvalidSort      = "sort must be name, quantity, or status"
	MsgInvalidDirection = "direction must be ascending or descending"
)

var SeedItems = []Item{
	{Name: "Access badge", Quantity: 240, Status: StatusInStock},
	{Name: "Docking station", Quantity: 12, Status: StatusLow},
	{Name: "Laptop sleeve", Quantity: 0, Status: StatusOutOfStock},
	{Name: "Monitor arm", Quantity: 58, Status: StatusInStock},
	{Name: "Noise-cancelling headset", Quantity: 4, Status: StatusLow},
	{Name: "Webcam", Quantity: 31, Status: StatusInStock},
}
