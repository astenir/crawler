package doubangroup

import (
	"fmt"
	"regexp"
	"time"

	"github.com/astenir/crawler/collect"
)

const urlListRe = `(https://www.douban.com/group/topic/[0-9a-z]+/)"[^>]*>([^<]+)</a>`
const ContentRe = `<div class="topic-content">[\s\S]*?好[\s\S]*?<div class="aside">`

var DoubangroupTask = &collect.Task{
	Name:     "find_douban_sun_room",
	WaitTime: 1 * time.Second,
	MaxDepth: 5,
	Cookie:   ``,
	Rule: collect.RuleTree{
		Root: func() []*collect.Request {
			var roots []*collect.Request
			for i := 0; i < 25; i += 25 {
				str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d", i)
				roots = append(roots, &collect.Request{
					Priority: 1,
					Url:      str,
					Method:   "GET",
					RuleName: "解析网站URL",
				})
			}
			return roots
		},
		Trunk: map[string]*collect.Rule{
			"解析网站URL": {ParseFunc: ParseURL},
			"解析阳台房":   {ParseFunc: GetSunRoom},
		},
	},
}

func ParseURL(ctx *collect.Context) collect.ParseResult {
	re := regexp.MustCompile(urlListRe)

	matches := re.FindAllSubmatch(ctx.Body, -1)
	result := collect.ParseResult{}

	for _, m := range matches {
		u := string(m[1])
		result.Requests = append(
			result.Requests, &collect.Request{
				Method:   "GET",
				Task:     ctx.Req.Task,
				Url:      u,
				Depth:    ctx.Req.Depth + 1,
				RuleName: "解析阳台房",
			})
	}
	return result
}

func GetSunRoom(ctx *collect.Context) collect.ParseResult {
	re := regexp.MustCompile(ContentRe)

	ok := re.Match(ctx.Body)
	if !ok {
		return collect.ParseResult{
			Items: []interface{}{},
		}
	}
	result := collect.ParseResult{
		Items: []interface{}{ctx.Req.Url},
	}
	return result
}

// func ParseURL(contents []byte, req *collect.Request) collect.ParseResult {
// 	re := regexp.MustCompile(urlListRe)

// 	matches := re.FindAllSubmatch(contents, -1)
// 	result := collect.ParseResult{}

// 	for _, m := range matches {
// 		u := string(m[1])
// 		result.Requests = append(
// 			result.Requests, &collect.Request{
// 				Method: "GET",
// 				Task:   req.Task,
// 				Url:    u,
// 				Depth:  req.Depth + 1,
// 				ParseFunc: func(c []byte, request *collect.Request) collect.ParseResult {
// 					return GetContent(c, u)
// 				},
// 			})
// 	}
// 	return result
// }

// func GetContent(contents []byte, url string) collect.ParseResult {
// 	re := regexp.MustCompile(ContentRe)

// 	ok := re.Match(contents)
// 	if !ok {
// 		return collect.ParseResult{
// 			Items: []interface{}{},
// 		}
// 	}

// 	result := collect.ParseResult{
// 		Items: []interface{}{url},
// 	}

// 	return result
// }
