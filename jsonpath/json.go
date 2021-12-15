/*
 Operator	    Supported	Description
       $         Y	         json的根节点. 通常在表达式开始位置.
       @         Y           The current node being processed by a filter predicate.
       *         X           Wildcard. Available anywhere a name or numeric are required.
       ..        X           Deep scan. Available anywhere a name is required.
       .         Y           Dot-notated child
  ['' (, '')]    X           Bracket-notated child or children
  [ (, )]        Y           Array index or indexes
  [start:end]    Y           Array slice operator
  [?()]          Y           Filter expression. Expression must evaluate to a boolean value.
 */
package jsonpath

import (
	"encoding/json"
	"fmt"
	"github.com/oliveagle/jsonpath"
	"reflect"
)

// 根据路径找到指定的值
func pickValByPath(dataStr string, path string) (interface{}, error){
	// 数据准备把字符串转换到对象内存储
	var json_data interface{}
	err := json.Unmarshal([]byte(dataStr), &json_data)
	if err != nil {
		return nil, err
	}

	res, err := jsonpath.JsonPathLookup(json_data, path)
	if err != nil {
		return nil, err
	}
	fmt.Printf("step 1 res: %v\n", path)
	fmt.Println(res)
	fmt.Println(reflect.TypeOf(res))
	return res, nil
}

// 条件过滤
func filterByPredicate(dataStr string, filterStr string)  {
	var json_data interface{}
	err := json.Unmarshal([]byte(dataStr), &json_data)
	if err != nil {
		fmt.Println(err)
		return
	}

	//or reuse lookup pattern
	pat, err := jsonpath.Compile(filterStr)
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := pat.Lookup(json_data)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("step 2 res:")
	fmt.Println(res)
}