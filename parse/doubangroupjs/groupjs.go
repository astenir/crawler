package doubangroupjs

import (
	"github.com/astenir/crawler/spider"
)

var DoubangroupJSTask = &spider.TaskModle{
	Property: spider.Property{
		Name:     "js_find_douban_sun_room",
		WaitTime: 2,
		MaxDepth: 5,
		Cookie:   ``,
	},
	Root: `
		var arr = new Array();
 		for (var i = 25; i <= 25; i+=25) {
			var obj = {
			   URL: "https://www.douban.com/group/szsh/discussion?start=" + i,
			   Priority: 1,
			   RuleName: "解析网站URL",
			   Method: "GET",
		   };
			arr.push(obj);
		};
		console.log(arr[0].URL);
		AddJsReq(arr);
			`,
	Rules: []spider.RuleModle{
		{
			Name: "解析网站URL",
			ParseFunc: `
			ctx.ParseJSReg("解析阳台房","(https://www.douban.com/group/topic/[0-9a-z]+/)\"[^>]*>([^<]+)</a>");
			`,
		},
		{
			Name: "解析阳台房",
			ParseFunc: `
			//console.log("parse output");
			ctx.OutputJS("<div class=\"topic-content\">[\\s\\S]*?阳台[\\s\\S]*?<div class=\"aside\">");
			`,
		},
	},
}
