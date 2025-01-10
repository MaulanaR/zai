package prompt

import (
	"fmt"
	"time"
)

func PromptDetermineAPIEndpoint() string {
	return fmt.Sprint(`Today is %s,
Check available data in current context first:
1. Review data already provided in Assistant role
2. Compare with required data fields

Then determine if new API data is needed:
1. If ANY required data is missing from context, specify the endpoint:
- contacts: Customer, Vendor, Employee queries
- sales_invoices: Sales Invoice queries 
- products: Product queries
- purchases_invoices: Purchase Invoice queries
- dashboards/profit_loss_simple: Profit and loss queries
- dashboards/balance_sheet_simple: Balance sheet queries
- dashboards/daily_sales: Daily sales data queries

2. If ALL required data is already available in context, respond with "endpoint": "null"

Consider data complete if Assistant role contains:
- Related database records
- Required fields for the analysis
- Data within valid time range

Respond only with the JSON decision object:
{"endpoint": "endpoint_name"}`, time.Now().Format("2006-01-02"))
}

func DefaultPromptRules() string {
	return fmt.Sprint(`Today is %s, Determine params for the endpoint. default params is {"is_skip_pagination":"true"}
if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}
Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`, time.Now().Format("2006-01-02"))
}

func PromptSalesInvoiceRules() string {
	return fmt.Sprint(`Today is %s, Determine params for the endpoint. default params {"is_skip_pagination":"true"}
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
{"params": {"param_key":"param_value"}}`, time.Now().Format("2006-01-02"))
}

func PromptPurchaseInvoiceRules() string {
	return fmt.Sprint(`Today is %s, Determine params for the endpoint. default params {"is_skip_pagination":"true"}
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
{"params": {"param_key":"param_value"}}`, time.Now().Format("2006-01-02"))
}

func PromptProductRules() string {
	return fmt.Sprint(`Today is %s, Determine params for the endpoint. default params {"is_skip_pagination":"true"}
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
	return fmt.Sprint(`Today is %s, Determine params for the endpoint. default params {"is_skip_pagination":"true"}
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
{"params": {"param_key":"param_value"}}`, time.Now().Format("2006-01-02"))
}
