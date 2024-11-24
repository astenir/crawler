package doubanbook

import (
	"regexp"
	"strconv"

	"github.com/astenir/crawler/collect"
	"go.uber.org/zap"
)

var DoubanBookTask = &collect.Task{
	Property: collect.Property{
		Name:     "douban_book_list",
		WaitTime: 2,
		MaxDepth: 5,
		Cookie:   `bid=pAGJ7yRQq5s; douban-fav-remind=1; _pk_id.100001.8cb4=967bd3913d4c4256.1723122696.; _pk_ref.100001.8cb4=%5B%22%22%2C%22%22%2C1729326734%2C%22https%3A%2F%2Fwww.baidu.com%2Fs%3Fwd%3D%E5%8D%83%E5%B9%B4%E7%8E%8B%E5%9B%BD%E7%9A%84%E5%85%AC%E4%B8%BB%26rsv_spt%3D1%26rsv_iqid%3D0x8ed9c74c0075489d%26issp%3D1%26f%3D8%26rsv_bp%3D1%26rsv_idx%3D2%26ie%3Dutf-8%26rqlang%3Dcn%26tn%3Dbaiduhome_pg%26rsv_enter%3D1%26rsv_dl%3Dtb%26oq%3D%25E5%258D%2583%25E5%25B9%25B4%25E7%258E%258B%25E5%259B%25BD%25E7%259A%2584%25E5%25A7%25AC%25E5%2590%259B%26rsv_btype%3Dt%26inputT%3D1137%26rsv_t%3D4d23TIgmY5871f19Ad4ZcpbuYeifb9FKhn12Dku5CLv0dUVeAoVjXtsu01TahtyBlNSv%26rsv_sug3%3D24%26rsv_sug1%3D18%26rsv_sug7%3D100%26rsv_pq%3Dd79c535900c7d269%26rsv_sug2%3D0%26rsv_sug4%3D2080%22%5D; _ga=GA1.1.1539337777.1731915190; _ga_RXNMP372GL=GS1.1.1731915190.1.0.1731915192.58.0.0; viewed="30137806_1007305"; dbcl2="258704165:wml2ul9TLEA"; push_noty_num=0; push_doumail_num=0; __utmv=30149280.25870; ck=ZPe_; __utma=30149280.1434693163.1723122696.1732188864.1732204686.9; __utmc=30149280; __utmz=30149280.1732204686.9.6.utmcsr=cn.bing.com|utmccn=(referral)|utmcmd=referral|utmcct=/; __utmt=1; __utmb=30149280.2.10.1732204686`,
	},
	Rule: collect.RuleTree{
		Root: func() ([]*collect.Request, error) {
			roots := []*collect.Request{
				{
					Priority: 1,
					Url:      "https://book.douban.com",
					Method:   "GET",
					RuleName: "数据tag",
				},
			}
			return roots, nil
		},
		Trunk: map[string]*collect.Rule{
			"数据tag": {ParseFunc: ParseTag},
			"书籍列表":  {ParseFunc: ParseBookList},
			"书籍简介": {
				ItemFields: []string{
					"书名",
					"作者",
					"页数",
					"出版社",
					"得分",
					"价格",
					"简介",
				},
				ParseFunc: ParseBookDetail,
			},
		},
	},
}

const regexpStr = `<a href="([^"]+)" class="tag">([^<]+)</a>`

func ParseTag(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(regexpStr)

	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		result.Requests = append(
			result.Requests, &collect.Request{
				Method:   "GET",
				Task:     ctx.Req.Task,
				Url:      "https://book.douban.com" + string(m[1]),
				Depth:    ctx.Req.Depth + 1,
				RuleName: "书籍列表",
			})
	}

	zap.S().Debugln("parse book tag,count:", len(result.Requests))
	// 在添加limit之前，临时减少抓取数量,防止被服务器封禁
	// result.Requests = result.Requests[:1]
	return result, nil
}

const BooklistRe = `<a.*?href="([^"]+)" title="([^"]+)"`

func ParseBookList(ctx *collect.Context) (collect.ParseResult, error) {
	re := regexp.MustCompile(BooklistRe)
	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}
	for _, m := range matches {
		req := &collect.Request{
			Priority: 100,
			Method:   "GET",
			Task:     ctx.Req.Task,
			Url:      string(m[1]),
			Depth:    ctx.Req.Depth + 1,
			RuleName: "书籍简介",
		}
		req.TmpData = &collect.Temp{}
		req.TmpData.Set("book_name", string(m[2]))
		result.Requests = append(result.Requests, req)
	}
	// 在添加limit之前，临时减少抓取数量,防止被服务器封禁
	// result.Requests = result.Requests[:3]
	zap.S().Debugln("parse book list,count:", len(result.Requests))

	return result, nil
}

var autoRe = regexp.MustCompile(`<span class="pl"> 作者</span>:[\d\D]*?<a.*?>([^<]+)</a>`)
var public = regexp.MustCompile(`<span class="pl">出版社:</span>[\d\D]*?<a.*?>([^<]+)</a>`)
var pageRe = regexp.MustCompile(`<span class="pl">页数:</span> ([^<]+)<br/>`)
var priceRe = regexp.MustCompile(`<span class="pl">定价:</span>([^<]+)<br/>`)
var scoreRe = regexp.MustCompile(`<strong class="ll rating_num " property="v:average">([^<]+)</strong>`)
var intoRe = regexp.MustCompile(`<div class="intro">[\d\D]*?<p>([^<]+)</p></div>`)

func ParseBookDetail(ctx *collect.Context) (collect.ParseResult, error) {
	bookName := ctx.Req.TmpData.Get("book_name")
	page, _ := strconv.Atoi(ExtraString(ctx.Body, pageRe))

	book := map[string]interface{}{
		"书名":  bookName,
		"作者":  ExtraString(ctx.Body, autoRe),
		"页数":  page,
		"出版社": ExtraString(ctx.Body, public),
		"得分":  ExtraString(ctx.Body, scoreRe),
		"价格":  ExtraString(ctx.Body, priceRe),
		"简介":  ExtraString(ctx.Body, intoRe),
	}
	data := ctx.Output(book)

	result := collect.ParseResult{
		Items: []interface{}{data},
	}
	zap.S().Debugln("parse book detail", data)

	return result, nil
}

func ExtraString(contents []byte, re *regexp.Regexp) string {

	match := re.FindSubmatch(contents)

	if len(match) >= 2 {
		return string(match[1])
	} else {
		return ""
	}
}
