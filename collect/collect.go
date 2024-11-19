package collect

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/astenir/crawler/proxy"
	"go.uber.org/zap"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type Fetcher interface {
	Get(req *Request) ([]byte, error)
}

type BaseFetch struct {
}

type BrowserFetch struct {
	Timeout time.Duration
	Proxy   proxy.ProxyFunc
	Logger  *zap.Logger
}

// DeterminEncoding 用于确定给定 bufio.Reader 中文本的编码类型。
// 该函数通过读取文本的前1024个字节来推断其编码格式。
func DeterminEncoding(r *bufio.Reader) encoding.Encoding {
	// 读取前1024个字节
	bytes, err := r.Peek(1024)

	// 如果读取过程中发生错误，打印错误信息并返回默认编码 UTF-8
	if err != nil {
		fmt.Printf("fetch error:%v", err)
		return unicode.UTF8
	}

	// 使用 charset 包确定编码格式，这里忽略了可能的错误和确定编码所需的信心值
	e, _, _ := charset.DetermineEncoding(bytes, "")
	// 返回确定的编码格式
	return e
}

// Get 基本访问
// 接受一个URL 方法用于发送GET请求并获取响应作为输入，返回响应的字节切片和可能出现的错误。
func (BaseFetch) Get(req *Request) ([]byte, error) {
	// 发送GET请求到指定的URL。
	resp, err := http.Get(req.Url)
	// 如果发生错误，打印错误信息并返回。
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// 确保在函数返回前关闭响应体。
	defer resp.Body.Close()

	// 检查HTTP状态码是否为200，表示请求成功。
	if resp.StatusCode != http.StatusOK {
		// 如果状态码不是200，打印错误信息并返回。
		fmt.Printf("Error status code:%d", resp.StatusCode)
		return nil, err
	}

	// 使用bufio.NewReader创建一个读取响应体的缓冲器。
	bodyReader := bufio.NewReader(resp.Body)
	// 确定响应体的编码方式。
	e := DeterminEncoding(bodyReader)
	// 使用确定的编码方式创建一个转换读取器，以便正确读取响应体。
	transReader := transform.NewReader(bodyReader, e.NewDecoder())
	// 读取并返回整个响应体。
	return io.ReadAll(transReader)
}

// 模拟浏览器访问
func (b BrowserFetch) Get(request *Request) ([]byte, error) {

	client := &http.Client{
		Timeout: b.Timeout,
	}

	if b.Proxy != nil {
		transport := http.DefaultTransport.(*http.Transport)
		transport.Proxy = b.Proxy
		client.Transport = transport
	}

	req, err := http.NewRequest("GET", request.Url, nil)
	if err != nil {
		return nil, fmt.Errorf("get url failed:%v", err)
	}

	// 设置Cookie
	if len(request.Cookie) > 0 {
		req.Header.Set("Cookie", request.Cookie)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	bodyReader := bufio.NewReader(resp.Body)
	e := DeterminEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return io.ReadAll(utf8Reader)
}
