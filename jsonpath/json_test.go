package jsonpath

import (
	"reflect"
	"testing"
)

var dataStr = `
{
    "store": {
        "book": [
            {
                "category": "reference",
                "author": "Nigel Rees",
                "title": "Sayings of the Century",
                "price": 8.95
            },
            {
                "category": "fiction",
                "author": "Evelyn Waugh",
                "title": "Sword of Honour",
                "price": 12.99
            },
            {
                "category": "fiction",
                "author": "Herman Melville",
                "title": "Moby Dick",
                "isbn": "0-553-21311-3",
                "price": 8.99
            },
            {
                "category": "fiction",
                "author": "J. R. R. Tolkien",
                "title": "The Lord of the Rings",
                "isbn": "0-395-19395-8",
                "price": 22.99
            }
        ],
        "bicycle": {
            "color": "red",
            "price": 19.95
        }
    },
    "expensive": 10
}
`

func Test_pickIntVal(t *testing.T) {
	v, err := pickValByPath(dataStr, "$.expensive") // 存在
	if err != nil {
		t.Logf("Error: %v", err)
	}
	t.Logf("$.expensive: %v, type: %v", v, reflect.TypeOf(v))

	v, err = pickValByPath(dataStr, "$.expensive123") // 不存在
	if err != nil {
		t.Logf("Error: %v", err)
	}
	t.Logf("$.expensive: %v, type: %v", v, reflect.TypeOf(v))

	v, err = pickValByPath(dataStr, "$.store.book") // 多层路径，获取到一个数组
	if err != nil {
		t.Logf("Error: %v", err)
	}
	t.Logf("$.expensive: %v, type: %v", v, reflect.TypeOf(v))
}

func Test_filterByPredicate(t *testing.T) {
	filter := func(dataStr string, filterStr string) {
		res, err := filterByPredicate(dataStr, filterStr)
		if err != nil {
			t.Error(err)
		}
		switch data := res.(type) {
		case []interface{}:
			t.Logf("filter Type:[]interface{}, len: %d. filter:%v",
				len(data), filterStr)
			for k, v := range data {
				t.Log("   ", k, ":", v)
			}
		default:
			t.Logf("filter Type: %v, res: %v, filter:%v", reflect.TypeOf(res), data, filterStr)
		}
	}


	filter(dataStr,
		`$.expensive`) // 10
	filter(dataStr,
		`$.store.book[0].price`) // 8.95
	filter(dataStr,
		`$.store.book[-1].isbn`) // 	"0-395-19395-8"
	filter(dataStr,
		`$.store.book[0,1].price`) // [8.95, 12.99]
	filter(dataStr,
		`$.store.book[0:2].price`) // [8.95, 12.99, 8.99]
	filter(dataStr,
		`$.store.book[:].price`)  	//全部：[8.9.5, 12.99, 8.9.9, 22.99]
	filter(dataStr,
		`$.store.book[?(@.price > 10)].title`) 	//["Sword of Honour", "The Lord of the Rings"]
	filter(dataStr,
		`$.store.book[?(@.author =~ /(?i).*REES/)].author`) // 正则


	// 失败，原因暂时未知
	t.Log("---------------------------------")
	filter(dataStr,
		`$.store.book[?(@.isbn)].price]`) // [8.99, 22.99] (ISBN非空的)
	filter(dataStr,
	`$.store.book[?(@.price < $.expensive)].price`)
}