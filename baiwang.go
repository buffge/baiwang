package baiwang

import (
	"bytes"
	b64 "encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/beevik/etree"
	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"
)

const (
	commonApiUrl                           = "http://tccdzfp.shfapiao.cn/zpdzfp"
	getAccessTokenUrl                      = "http://sc.bwfapiao.com/fpserver/FpServlet"
	getPdfUrl                              = getAccessTokenUrl
	successRetCode                         = "0000"
	TaxTypeCommon         TaxType          = "0" // 普通征税
	TaxTypeReduce         TaxType          = "1" // 减按计征
	TaxTypeDisparity      TaxType          = "2" // 差额征税
	InvoiceTypeBlue       InvoiceType      = "0" // 蓝字发票
	InvoiceTypeRed        InvoiceType      = "1" // 红字发票
	PreferentialTypeFalse PreferentialType = "0" // 没有使用优惠政策
	PreferentialTypeTrue  PreferentialType = "1" // 使用了优惠政策
	TaxLineTypeNormal     TaxLineType      = "0" // 正常行
	TaxLineTypeDeduct     TaxLineType      = "1" // 折扣行
	TaxLineTypeBeDeducted TaxLineType      = "2" // 被折扣行
	defaultTaxRate                         = 0.06
)

var (
	notValidResp         = errors.New("返回值不是正确的值")
	defaultCharsetRender = func(c string, i io.Reader) (io.Reader, error) {
		return charset.NewReaderLabel(c, i)
	}
)

type (
	Client struct {
		AppID      string
		AppKey     string
		SellerName string
	}
	TaxType          string
	InvoiceType      string
	PreferentialType string
	TaxLineType      string
	// 电子发票开具请求数据
	ApplyElectronicInvoiceData struct {
		ReqID                 string
		TaxType               TaxType     // 征税方式
		InvoiceType           InvoiceType // 开票类型
		SellerTaxpayerID      string      // 销售方纳税人识别号
		SellerName            string      // 销售方名称
		SellerAddressAndPhone string      // 销售方地址、电话
		SellerBankNumber      string      // 销售方银行账号
		BuyerName             string      // 购买方名称
		Drawer                string      // 开票人
		TotalFee              float64     // 价税合计 单位:元
		GoodTotalFee          float64     // 商品价格 单位:元
		TaxTotalFee           float64     // 税金额 单位:元
		CodeTableVersion      string      // 编码表版本号
		TaxLineType           TaxLineType
		// FPHXZ    发票行性质    1    是    0正常行、1折扣行、2被折扣行
		ProjectName  string           // 项目名称
		ProjectFee   float64          // 项目金额
		TaxRate      float64          // 税率
		TaxTee       float64          // 税额
		GoodCode     string           // 商品编码
		Preferential PreferentialType // 是否使用了优惠政策
	}
	// 发票信息
	Invoice struct {
		ReqID      string
		Code       string
		Number     string
		Date       time.Time
		DeviceID   string
		Secret     string
		VerifyCode string
		Backup     string
	}
)

func NewClient(appID, appKey, sellerName string) *Client {
	return &Client{
		AppID:      appID,
		AppKey:     appKey,
		SellerName: sellerName,
	}
}
func generateApplyElectronicInvoiceXml(d *ApplyElectronicInvoiceData) *etree.Document {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="gbk"`)
	doc.Indent(0)
	business := doc.CreateElement("business")
	business.CreateAttr("id", "FPKJ")
	business.CreateAttr("comment", "发票开具")
	temp1 := business.CreateElement("HTJS_DZFPKJ")
	temp1.CreateAttr("class", "HTJS_DZFPKJ")
	temp2 := temp1.CreateElement("COMMON_FPKJ_FPT")
	temp2.CreateAttr("class", "COMMON_FPKJ_FPT")
	reqID := temp2.CreateElement("FPQQLSH")
	reqID.SetText(d.ReqID)
	invoiceType := temp2.CreateElement("KPLX")
	invoiceType.SetText(string(d.InvoiceType))
	taxType := temp2.CreateElement("ZSFS")
	taxType.SetText(string(d.TaxType))
	sellerTaxpayerID := temp2.CreateElement("XSF_NSRSBH")
	sellerTaxpayerID.SetText(d.SellerTaxpayerID)
	sellerName := temp2.CreateElement("XSF_MC")
	sellerName.SetText(d.SellerName)
	sellerAddressAndPhone := temp2.CreateElement("XSF_DZDH")
	sellerAddressAndPhone.SetText(d.SellerAddressAndPhone)
	sellerBankNumber := temp2.CreateElement("XSF_YHZH")
	sellerBankNumber.SetText(d.SellerBankNumber)
	temp2.CreateElement("XSF_LXFS")
	temp2.CreateElement("GMF_NSRSBH")
	buyerName := temp2.CreateElement("GMF_MC")
	buyerName.SetText(d.BuyerName)
	temp2.CreateElement("GMF_DZDH")
	temp2.CreateElement("GMF_YHZH")
	temp2.CreateElement("GMF_LXFS")
	drawer := temp2.CreateElement("KPR")
	drawer.SetText(d.Drawer)
	temp2.CreateElement("SKR")
	temp2.CreateElement("FHR")
	temp2.CreateElement("YFP_DM")
	temp2.CreateElement("YFP_HM")
	totalFee := temp2.CreateElement("JSHJ")
	totalFee.SetText(fmt.Sprintf("%.2f", d.TotalFee))
	goodTotalFee := temp2.CreateElement("HJJE")
	goodTotalFee.SetText(fmt.Sprintf("%.2f", d.GoodTotalFee))
	taxTotalFee := temp2.CreateElement("HJSE")
	taxTotalFee.SetText(fmt.Sprintf("%.2f", d.TaxTotalFee))
	temp2.CreateElement("KCE")
	temp2.CreateElement("BZ")
	codeTableVersion := temp2.CreateElement("BMB_BBH")
	codeTableVersion.SetText(d.CodeTableVersion)
	temp3 := temp1.CreateElement("COMMON_FPKJ_XMXXS")
	temp3.CreateAttr("class", "COMMON_FPKJ_XMXX")
	temp3.CreateAttr("size", "1")
	temp4 := temp3.CreateElement("COMMON_FPKJ_XMXX")
	taxLineType := temp4.CreateElement("FPHXZ")
	taxLineType.SetText(string(d.TaxLineType))
	goodCode := temp4.CreateElement("SPBM")
	goodCode.SetText(d.GoodCode)
	temp4.CreateElement("ZXBM")
	preferential := temp4.CreateElement("YHZCBS")
	preferential.SetText(string(d.Preferential))
	temp4.CreateElement("LSLBS")
	temp4.CreateElement("ZZSTSGL")
	projectName := temp4.CreateElement("XMMC")
	projectName.SetText(d.ProjectName)
	temp4.CreateElement("GGXH")
	temp4.CreateElement("DW")
	projectCount := temp4.CreateElement("XMSL")
	projectCount.SetText("1")
	projectItemFee := temp4.CreateElement("XMDJ")
	projectItemFee.SetText(fmt.Sprintf("%.2f", d.ProjectFee))
	projectFee := temp4.CreateElement("XMJE")
	projectFee.SetText(fmt.Sprintf("%.2f", d.ProjectFee))
	taxRate := temp4.CreateElement("SL")
	taxRate.SetText(fmt.Sprintf("%.2f", d.TaxRate))
	taxTee := temp4.CreateElement("SE")
	taxTee.SetText(fmt.Sprintf("%.2f", d.TaxTee))
	return doc
}
func generateApplyElectronicInvoicePostXml(d *ApplyElectronicInvoiceData,
	oriXML *etree.Document) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="gbk"`)
	doc.Indent(0)
	business := doc.CreateElement("business")
	business.CreateAttr("comment", "电子发票开具")
	business.CreateAttr("id", "DZFPKJ")
	body := business.CreateElement("body")
	body.CreateAttr("yylxdm", "1")
	input := body.CreateElement("input")
	reqID := input.CreateElement("DJBH")
	reqID.CreateText(d.ReqID)
	actData := input.CreateElement("FPXML")
	var actDataBytes []byte
	var err error
	if actDataBytes, err = oriXML.WriteToBytes(); err != nil {
		return nil, err
	}
	actData.CreateText(b64.StdEncoding.EncodeToString(actDataBytes))
	return doc, nil
}

func (c *Client) GetAccessToken() (string, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="gbk"`)
	doc.Indent(0)
	business := doc.CreateElement("business")
	business.CreateAttr("id", "YHYZ")
	business.CreateAttr("comment", "用户验证")
	user := business.CreateElement("user")
	user.CreateAttr("lxdm", "用户类型")
	name := user.CreateElement("name")
	name.SetText(c.AppID)
	sn := user.CreateElement("sn")
	sn.SetText(c.AppKey)
	buf := new(bytes.Buffer)
	_, _ = doc.WriteTo(buf)
	resp, err := http.Post(getAccessTokenUrl, "text/plain", buf)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	resDoc := etree.NewDocument()
	resDoc.ReadSettings.CharsetReader = defaultCharsetRender
	if _, err := resDoc.ReadFrom(resp.Body); err != nil {
		return "", err
	}
	businessEle := resDoc.SelectElement("business")
	if businessEle == nil {
		return "", notValidResp
	}
	retCodeData := businessEle.SelectElement("returnCode")
	if retCodeData == nil {
		return "", notValidResp
	}
	if retCodeData.Text() != successRetCode {
		return "", errors.New(businessEle.SelectElement("returnMsg").Text())
	}
	return businessEle.SelectElement("user").SelectElement("access_token").Text(), nil
}
func (c *Client) ApplyElectronicInvoice(data *ApplyElectronicInvoiceData) (invoice *Invoice,
	err error) {
	totalFee := 1.0
	taxRate := 0.06
	goodTotalFee := totalFee / (1 + taxRate)
	data.SellerTaxpayerID = c.AppID
	data.SellerName = c.SellerName
	data.GoodTotalFee = goodTotalFee
	data.TaxTotalFee = totalFee - goodTotalFee
	data.ProjectFee = goodTotalFee
	if data.TaxRate == 0 {
		data.TaxRate = defaultTaxRate
	}
	data.TaxTee = totalFee - goodTotalFee
	oriXML := generateApplyElectronicInvoiceXml(data)
	doc, err := generateApplyElectronicInvoicePostXml(data, oriXML)
	if err != nil {
		return nil, err
	}
	bts, _ := doc.WriteToBytes()
	buf := new(bytes.Buffer)
	buf.WriteString(b64.StdEncoding.EncodeToString(bts))
	resp, err := http.Post(commonApiUrl, "text/plain", buf)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respB64Data, _ := ioutil.ReadAll(resp.Body)
	respData := make([]byte, b64.StdEncoding.DecodedLen(len(respB64Data)))
	_, _ = b64.StdEncoding.Decode(respData, respB64Data)
	respData = trimRightBytes(respData)
	resDoc := etree.NewDocument()
	resDoc.ReadSettings.CharsetReader = defaultCharsetRender
	if err := resDoc.ReadFromBytes(respData); err != nil {
		return nil, err
	}
	businessEle := resDoc.SelectElement("business")
	if businessEle == nil {
		return nil, notValidResp
	}
	invoiceDetailEle := businessEle.SelectElement("HTJS_DZFPKJ")
	retCodeEle := invoiceDetailEle.SelectElement("RETURNCODE")
	if retCodeEle == nil {
		return nil, notValidResp
	}
	if retCodeEle.Text() != successRetCode {
		return nil, errors.New(invoiceDetailEle.
			SelectElement("RETURNMSG").Text())
	}
	invoice = &Invoice{
		Code:       invoiceDetailEle.SelectElement("FP_DM").Text(),
		Number:     invoiceDetailEle.SelectElement("FP_HM").Text(),
		ReqID:      invoiceDetailEle.SelectElement("FPQQLSH").Text(),
		Secret:     invoiceDetailEle.SelectElement("FP_MW").Text(),
		DeviceID:   invoiceDetailEle.SelectElement("JQBH").Text(),
		VerifyCode: invoiceDetailEle.SelectElement("JYM").Text(),
		Backup:     invoiceDetailEle.SelectElement("BZ").Text(),
		Date:       time.Time{},
	}
	invoice.Date, _ = time.Parse("20060102150405", invoiceDetailEle.SelectElement("KPRQ").Text())
	return invoice, nil
}
func (c *Client) DownloadElectronicInvoice(ak string, invoice *Invoice) (url string, err error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="gbk"`)
	doc.Indent(0)
	business := doc.CreateElement("business")
	business.CreateAttr("FPCFDZ", "发票存放地址")
	business.CreateAttr("id", "FPCFDZ")
	user := business.CreateElement("user")
	user.CreateAttr("lxdm", "用户类型")
	name := user.CreateElement("name")
	name.SetText(c.AppID)
	accessToken := user.CreateElement("access_token")
	accessToken.SetText(ak)
	temp1 := business.CreateElement("COMMON_FPXX_CFDZS")
	temp1.CreateAttr("size", "存放地址数量")
	temp2 := temp1.CreateElement("COMMON_FPXX_CFDZ")
	invoiceCode := temp2.CreateElement("FP_DM")
	invoiceCode.SetText(invoice.Code)
	invoiceNumber := temp2.CreateElement("FP_HM")
	invoiceNumber.SetText(invoice.Number)
	totalFee := temp2.CreateElement("JSHJ")
	totalFee.SetText("1")
	date := temp2.CreateElement("KPRQ")
	date.SetText(invoice.Date.Format("2006010215"))
	buf := new(bytes.Buffer)
	_, _ = doc.WriteTo(buf)
	resp, err := http.Post(getPdfUrl, "text/plain", buf)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	respData, _ := ioutil.ReadAll(resp.Body)
	resDoc := etree.NewDocument()
	resDoc.ReadSettings.CharsetReader = defaultCharsetRender
	if err := resDoc.ReadFromBytes(respData); err != nil {
		return "", err
	}
	businessEle := resDoc.SelectElement("business")
	if businessEle == nil {
		return "", notValidResp
	}
	retCodeEle := businessEle.SelectElement("returnCode")
	if retCodeEle == nil {
		return "", notValidResp
	}
	if retCodeEle.Text() != successRetCode {
		return "", errors.New(businessEle.
			SelectElement("returnMsg").Text())
	}
	url = businessEle.SelectElement("COMMON_FPXX_CFDZS").
		SelectElement("COMMON_FPXX_CFDZ").
		SelectElement("FP_URL").Text()
	return url, nil
}
