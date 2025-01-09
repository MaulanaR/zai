package prompt

import "fmt"

func PromptDetermineAPIEndpoint() string {
	return fmt.Sprint(`Determine if new API data is needed or if existing data suffices:
1. If new data is needed, specify the endpoint and parameters:
- contacts: Customer, Vendor, Employee queries
- sales_invoices: Sales Invoice queries
- products: Product queries
- purchases_invoices: Purchase Invoice queries
- dashboards/profit_loss_simple: Profit and loss queries
- dashboards/balance_sheet_simple: Balance sheet queries
- dashboards/daily_sales: Daily sales data queries

2. If existing data suffices, respond with "endpoint": "null".

Respond only with the JSON decision object:
{"endpoint": "endpoint_name"}`)
}

func DefaultPromptRules() string {
	return fmt.Sprint(`Determine params for the endpoint. default params is {"is_skip_pagination":"true"}
if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}
Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`)
}

func PromptSalesInvoiceRules() string {
	return fmt.Sprint(`Determine params for the endpoint. default params {"is_skip_pagination":"true"}
avaiable field for queries : 
customer.name, 
payment_status [enum : open, paid],
date,
time,
number,
description,
customer.name,
currency.name,
subtotal,
total_discount,
subtotal_before_tax,
total_tax,
total_cash_amount,
total_amount,
total_payment,
line_items as product information

if user need information about product, then add param includes[line_items]=true

if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}

Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`)
}

func PromptPurchaseInvoiceRules() string {
	return fmt.Sprint(`Determine params for the endpoint. default params {"is_skip_pagination":"true"}
available fields for queries:
description,
date,
time,
number,
note,
total_amount.

if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}

Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`)
}

func PromptProductRules() string {
	return fmt.Sprint(`Determine params for the endpoint. default params {"is_skip_pagination":"true"}
available fields for queries:
code,
name,
description,
category.name,
catalog.name,
quantity.on_hand,
quantity.on_order,
quantity.on_hold,
unit_price_gross,
unit_price,
unit_cogs.

if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}

Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`)
}

func PromptContactRules() string {
	return fmt.Sprint(`Determine params for the endpoint. default params {"is_skip_pagination":"true"}
available fields for queries:
name,
note,
national_id_number,
tax_id_number,
is_customer,
is_supplier,
is_employee,
is_salesman,
is_active,
customer_category.name

if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}

Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`)
}
