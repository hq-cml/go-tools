package jsonpath

import "testing"

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
	pickValByPath(dataStr, "$.expensive")
}

func Test_filterByPredicate(t *testing.T) {
	filterByPredicate(dataStr, `$.store.book[?(@.price < $.expensive)].price`)
	filterByPredicate(dataStr, `$.store.book[?(@.price > 10)].title`)
	filterByPredicate(dataStr, `$.store.book[0:2].price`)
	filterByPredicate(dataStr, `$.expensive`)  // 10
}