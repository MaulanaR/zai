package model

import (
	"grest.dev/grest"
)

type ContactResp struct {
	Data []Contact `json:"results"`
}
type Contact struct {
	Name                 grest.NullString `json:"name"`
	Note                 grest.NullText   `json:"note"`
	NationalIDNumber     grest.NullString `json:"national_id_number"`
	TaxIDNumber          grest.NullString `json:"tax_id_number"`
	IsCustomer           grest.NullBool   `json:"is_customer"`
	IsSupplier           grest.NullBool   `json:"is_supplier"`
	IsEmployee           grest.NullBool   `json:"is_employee"`
	IsSalesman           grest.NullBool   `json:"is_salesman"`
	IsActive             grest.NullBool   `json:"is_active"`
	CustomerCategoryName grest.NullString `json:"customer_category.name"`
	// TaxIDAddress         grest.NullString `json:"tax_id_address"`
	// BussinessIDNumber    grest.NullString `json:"bussiness_id_number"`
	// Addresses            grest.NullJSON   `json:"addresses"`
	// Phones               grest.NullJSON   `json:"phones"`
	// Emails               grest.NullJSON   `json:"emails"`
}

type SalesInvoicesResp struct {
	Data []SalesInvoiceDetail `json:"results"`
}
type SalesInvoiceDetail struct {
	Status        grest.NullString   `json:"status"`
	PaymentStatus grest.NullString   `json:"payment_status"`
	Date          grest.NullDate     `json:"date"`
	Time          grest.NullDateTime `json:"time"`
	Number        grest.NullString   `json:"number"`
	Description   grest.NullString   `json:"description"`
	CustomerName  grest.NullString   `json:"customer.name"`
	CurrencyName  grest.NullString   `json:"currency.name"`
	TotalAmount   grest.NullFloat64  `json:"total_amount"`
	TotalPayment  grest.NullFloat64  `json:"total_payment"`
	LineItems     []LineItems        `json:"line_items"`
	// Receivable              grest.NullFloat64  `json:"receivable"`
	// Balance                 grest.NullFloat64  `json:"balance"`
	// Subtotal                grest.NullFloat64  `json:"subtotal"`
	// TotalDiscount           grest.NullFloat64  `json:"total_discount"`
	// TotalDiscountPercentage grest.NullFloat64  `json:"total_discount_percentage"`
	// SubtotalBeforeTax       grest.NullFloat64  `json:"subtotal_before_tax"`
	// TotalTax                grest.NullFloat64  `json:"total_tax"`
	// TotalCashAmount         grest.NullFloat64  `json:"total_cash_amount"`
	// TotalOther              grest.NullFloat64  `json:"total_other"`
}
type LineItems struct {
	ProductCode         grest.NullString  `json:"product.code"`
	ProductName         grest.NullString  `json:"product.name"`
	ProductCategoryName grest.NullString  `json:"product.category.name"`
	UnitName            grest.NullString  `json:"unit.name"`
	Quantity            grest.NullFloat64 `json:"quantity"`
	UnitPrice           grest.NullFloat64 `json:"unit_price"`
	DiscountAmount      grest.NullFloat64 `json:"discount.amount"`
	Note                grest.NullString  `json:"note"`
	UnitCOGS            grest.NullFloat64 `json:"unit_cogs"`
}

type ProductResp struct {
	Data []Product `json:"results"`
}
type Product struct {
	Code            grest.NullString  `json:"code"`
	Name            grest.NullString  `json:"name"`
	Description     grest.NullString  `json:"description"`
	CategoryName    grest.NullUUID    `json:"category.name"`
	CatalogName     grest.NullString  `json:"catalog.name"`
	QuantityOnHand  grest.NullFloat64 `json:"quantity.on_hand"`
	QuantityOnOrder grest.NullFloat64 `json:"quantity.on_order"`
	QuantityOnHold  grest.NullFloat64 `json:"quantity.on_hold"`
	UnitPriceGross  grest.NullFloat64 `json:"unit_price_gross"`
	UnitPrice       grest.NullFloat64 `json:"unit_price"`
	UnitCogs        grest.NullFloat64 `json:"unit_cogs"`
	// UnitConversions grest.NullJSON    `json:"unit_conversions"`
	// SellingPrices   grest.NullJSON    `json:"selling_prices"`
	// SellingTaxes    grest.NullJSON    `json:"selling_taxes"`
	// PurchasingTaxes grest.NullJSON    `json:"purchasing_taxes"`
	// UnitCost        grest.NullFloat64 `json:"unit_cost"`
	// MinimumStock    grest.NullFloat64 `json:"minimum_stock"`
}

type PurchaseInvResp struct {
	Data []PurchaseInvDetail `json:"results"`
}
type PurchaseInvDetail struct {
	Description grest.NullString   `json:"description"`
	Date        grest.NullDate     `json:"date"`
	Time        grest.NullDateTime `json:"time"`
	Number      grest.NullString   `json:"number"`
	Note        grest.NullString   `json:"note"`
	TotalAmount grest.NullFloat64  `json:"total_amount"`

	// SupplierName   grest.NullString   `json:"supplier.name"`
	// DepartmentName grest.NullString   `json:"department.name"`
	// ProjectName    grest.NullString   `json:"project.name"`
	// CostCodeName   grest.NullString   `json:"cost_code.name"`
	// WarehouseName  grest.NullString   `json:"warehouse.name"`
}
