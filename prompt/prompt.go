package prompt

import (
	"fmt"
	"strings"
	"time"
)

// - dashboards/daily_sales: Daily sales data queries
func PromptDetermineAPIEndpoint() string {
	return fmt.Sprint(`You're very smart AI. Today is ` + time.Now().Format("2006-01-02") + `,
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

2. If ALL required data is already available in context, respond with "endpoint": "null"

Consider data complete if Assistant role contains:
- Related database records
- Required fields for the analysis
- Data within valid time range

Respond only with the JSON decision object:
{"endpoint": "endpoint_name"}`)
}

func DefaultPromptRules() string {
	return fmt.Sprint(`Today is ` + time.Now().Format("2006-01-02") + `, Determine params for the endpoint. default params is {"per_page":"10000"}
if user request for date filtering, use param date[$gte] or date[$lte] or date[$eq] with the format YYYY-MM-DD.
e.g : {"date[$gte]":"2000-01-25"}
Respond only with the JSON decision object:
{"params": {"param_key":"param_value"}}`)
}

func PromptSalesInvoiceRules() string {
	return fmt.Sprint(`Today is ` + time.Now().Format("2006-01-02") + `, Determine params for the endpoint. default params {"per_page":"10000"}
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
	return fmt.Sprint(`Today is ` + time.Now().Format("2006-01-02") + `, Determine params for the endpoint. default params {"per_page":"10000"}
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
	return fmt.Sprint(`Today is ` + time.Now().Format("2006-01-02") + `, Determine params for the endpoint. default params {"per_page":"10000"}
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
	return fmt.Sprint(`Today is ` + time.Now().Format("2006-01-02") + `, Determine params for the endpoint. default params {"per_page":"10000"}
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

func SystemMSG() string {
	output := fmt.Sprintf(`
<response_rules>
	<!-- Strict response format rules -->
	<api_request>
		When data is needed, respond ONLY with a JSON object containing endpoint and params.
		No explanation, no additional text.

		Format:
		{"input" : false, "endpoint": "endpoint_name", "params": {"param_key": "param_value" // Include all required parameters} }

		If no data already provided:
		{"input" : false, "endpoint": "null", "response" : "" //other response set null}
	</api_request>

	<data_input>
		When handling data input, respond ONLY with a JSON object.
		No explanation, no additional text.

		Format for contact (customer, supplier, employee):
		{"input" : true, "endpoint" : "contacts", "type": "kontak", "params" : {"name": "[nama]", "phone": "[phone]", "email": "[email]" , "is_customer" : boolean,"is_employee" : boolean,"is_supplier" : boolean}}

		Format for product:
		{"input" : true, "endpoint" : "products", "type": "produk", "params" : {"name": "[nama]", "price": "[harga]", "produk.category": "[kategori]"}}
	</data_input>
</response_rules>

<data_rules>
	<!-- Previous data rules remain the same -->
	<api_endpoints>
		<available_endpoints>
			- contacts: Customer, Vendor, Employee queries
			- sales_invoices: Sales Invoice queries
			- products: Product queries
			- purchases_invoices: Purchase Invoice queries
			- dashboards/profit_loss_simple: Profit Loss queries
			- dashboards/balance_sheet_simple: Balance Sheet queries
		</available_endpoints>

		<endpoint_params>
			<default_params>
				{"per_page": "10"}
				if endpoint is contacts, then {"per_page": "50"}
			</default_params>

			<available_fields>
				<sales_invoices>
					- customer.name
					- payment_status (enum: open, paid)
					- date
					- time
					- number
					- description
					- currency.name
					- subtotal
					- total_discount
					- subtotal_before_tax
					- total_tax
					- total_cash_amount
					- total_amount
					- total_payment
					- line_items (product information)
				</sales_invoices>

				<purchases_invoices>
					- description
					- date
					- time
					- number
					- note
					- total_amount
				</purchases_invoices>

				<products>
					- code
					- name
					- description
					- category.name
					- catalog.name
					- quantity.on_hand
					- quantity.on_order
					- quantity.on_hold
					- unit_price_gross
					- unit_price
					- unit_cogs
				</products>

				<contacts>
					- name
					- note
					- national_id_number
					- tax_id_number
					- is_customer
					- is_supplier
					- is_employee
					- is_salesman
					- is_active
					- customer_category.name
				</contacts>
			</available_fields>

			<special_params>
				<sales_invoices_query>
					if user need to show the products of sales invoices
					{"includes[line_items]": "true"}
				</sales_invoices_query>

				<date_query>
					Format: YYYY-MM-DD
					Operators: date[$gte], date[$lte], date[$eq]
				</date_query>
			</special_params>
		</endpoint_params>
	</api_endpoints>

	<data_input>
		<allowed_types>
			- contacts
			- products
		</allowed_types>

		<validation_rules>
			<contacts>
				Required fields:
				- name (string)
				- phone (string)
				- email (string, valid email format)
			</contacts>

			<products>
				Required fields:
				- name (string)
				- price (number)
				- category (string)
			</products>
		</validation_rules>
		<input_rules>
			- do not edit/add anything to fields already filled by the user
		</input_rules>
	</data_input>
</data_rules>

<processing_rules>
	1. MUST respond with clean JSON only
	2. NO explanatory text before or after JSON
	3. Check context before requesting data
	4. Include default params when needed
	5. Validate all required fields for input
	6. Use proper date format YYYY-MM-DD
</processing_rules>

<today_date>
` + time.Now().Format("2006-01-02") + `
</today_date>`)

	output = strings.NewReplacer("\n", " ", "\t", " ").Replace(output)
	return fmt.Sprint(output)
}

func GenerateResRule() string {
	output := fmt.Sprintf(`
	<today_date>
	` + time.Now().Format("2006-01-02") + `
	</today_date>
	<response_rules>
	- Jawab pertanyaan pengguna secara natural berdasarkan data yang diberikan
	- Hanya tampilkan data yang bisa dibaca manusia, jangan tampilkan data yang nilainya null/NULL
	- Hanya sertakan informasi yang relevan dan jangan menjawab jika pertanyaan tidak terkait dengan data yang ditentukan atau tidak tentang Zahir.
	- Format semua harga dalam mata uang Rupiah.
	- Respon dalam BAHASA INDONESIA
	- Jangan response dalam chart/grafik jika user tidak menginginkan
	- JIKA pengguna ingin disajikan dalam bentuk chart/grafik, maka GUNAKAN HIGHCHART DALAM HTML & JS
	- Default sajikan sebagai tabel html
	</response_rules>`)

	output = strings.NewReplacer("\n", " ", "\t", " ").Replace(output)
	return fmt.Sprint(output)
}
