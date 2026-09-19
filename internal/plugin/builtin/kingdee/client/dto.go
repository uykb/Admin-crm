package client

// AuthRequest 金蝶 ApiService.Authenticate 请求体
type AuthRequest struct {
	AcctID   string `json:"acctID"`
	UserName string `json:"username"`
	Password string `json:"password"`
	Lcid     int    `json:"lcid"`
}

// AuthResponse 金蝶 ApiService.Authenticate 响应体
type AuthResponse struct {
	LoginResultType int `json:"LoginResultType"`
	ResponseStatus  struct {
		IsSuccess bool `json:"IsSuccess"`
		Errors    []struct {
			Message string `json:"Message"`
		} `json:"Errors"`
	} `json:"ResponseStatus"`
	K3CloudToken string `json:"K3CloudToken"`
}

// BillQueryData 金蝶 ExecuteBillQuery 序列化数据
type BillQueryData struct {
	FormID       string `json:"FormId"`
	FieldKeys    string `json:"FieldKeys"`
	FilterString string `json:"FilterString,omitempty"`
	OrderString  string `json:"OrderString,omitempty"`
	TopRow       int    `json:"TopRow,omitempty"`
	Limit        int    `json:"Limit,omitempty"`
	StartRow     int    `json:"StartRow,omitempty"`
}

// BillQueryRequest 金蝶 ExecuteBillQuery 外层包装
type BillQueryRequest struct {
	Data BillQueryData `json:"data"`
}

// MaterialItem 物料/商品对象
type MaterialItem struct {
	ID            string `json:"id"`
	Number        string `json:"number"`
	Name          string `json:"name"`
	Specification string `json:"specification"`
	GroupName     string `json:"group_name"`
}

// CustomerItem 客户对象
type CustomerItem struct {
	ID        string `json:"id"`
	Number    string `json:"number"`
	Name      string `json:"name"`
	CustomerGroup string `json:"customer_group"`
}

// SalesOrderItem 销售订单对象
type SalesOrderItem struct {
	ID             string `json:"id"`
	BillNo         string `json:"bill_no"`
	Date           string `json:"date"`
	CustomerName   string `json:"customer_name"`
	BillTypeName   string `json:"bill_type_name"`
	DocumentStatus string `json:"document_status"`
}
