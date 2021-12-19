/*
 * 在不知道具体的json结构的时候，可以用json path来尝试取出值
 *
 * Operator	    Supported	Description
 *      $         Y	         json 的根节点. 通常在表达式开始位置.
 *      @         Y           The current node being processed by a filter predicate.
 *      *         X           Wildcard. Available anywhere a name or numeric are required.
 *      ..        X           Deep scan. Available anywhere a name is required.
 *      .         Y           Dot-notated child
 * ['' (, '')]    X           Bracket-notated child or children
 * [ (, )]        Y           Array index or indexes
 * [start:end]    Y           Array slice operator
 * [?()]          Y           Filter expression. Expression must evaluate to a boolean value.
 *
 * Note: golang 支持正则表达式标志，格式如 (?imsU)pattern
 */
package jsonpath

import (
	"encoding/json"
	"github.com/oliveagle/jsonpath"
)

// 根据路径找到指定的值
func pickValByPath(dataStr string, path string) (interface{}, error){
	// 数据准备把字符串转换到对象内存储
	var jsonData interface{}
	err := json.Unmarshal([]byte(dataStr), &jsonData)
	if err != nil {
		return nil, err
	}

	res, err := jsonpath.JsonPathLookup(jsonData, path)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// 条件过滤
func filterByPredicate(dataStr string, filterStr string) (interface{}, error) {
	var jsonData interface{}
	err := json.Unmarshal([]byte(dataStr), &jsonData)
	if err != nil {
		return nil, err
	}

	//or reuse lookup pattern
	pattern, err := jsonpath.Compile(filterStr)
	if err != nil {
		return nil, err
	}
	res, err := pattern.Lookup(jsonData)
	if err != nil {
		return nil, err
	}

	return res, nil
}