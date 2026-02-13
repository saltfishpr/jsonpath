package jsonpath_test

import (
	"fmt"

	"github.com/saltfishpr/jsonpath"
)

var rfcExampleJSON = `{
  "store": {
    "book": [
      {"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
      {"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
      {"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
      {"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99}
    ],
    "bicycle": {"color": "red", "price": 399}
  }
}`

func ExampleGetMany() {
	fmt.Println("1. the authors of all books in the store")
	r := jsonpath.GetMany(rfcExampleJSON, "$.store.book[*].author")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n2. all authors")
	r = jsonpath.GetMany(rfcExampleJSON, "$..author")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n3. all things in the store, which are some books and a red bicycle")
	r = jsonpath.GetMany(rfcExampleJSON, "$.store.*")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n4. the prices of everything in the store")
	r = jsonpath.GetMany(rfcExampleJSON, "$.store..price")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n5. the third book")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[2]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n6. the third book's author")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[2].author")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n7. empty result: the third book does not have a \"publisher\" member")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[2].publisher")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n8. the last book in order")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[-1]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n9. the first two books")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[:2]")
	for _, v := range r {
		fmt.Println(v.Value())
	}
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[0,1]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n10. all books with an ISBN number")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[?(@.isbn)]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n11. all books cheaper than 10")
	r = jsonpath.GetMany(rfcExampleJSON, "$..book[?(@.price<10)]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n12. all member values and array elements contained in the input value")
	r = jsonpath.GetMany(rfcExampleJSON, "$..*")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	// Output:
	// 1. the authors of all books in the store
	// Nigel Rees
	// Evelyn Waugh
	// Herman Melville
	// J. R. R. Tolkien
	//
	// 2. all authors
	// Nigel Rees
	// Evelyn Waugh
	// Herman Melville
	// J. R. R. Tolkien
	//
	// 3. all things in the store, which are some books and a red bicycle
	// [{"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95} {"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99} {"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99} {"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99}]
	// map[color:red price:399]
	//
	// 4. the prices of everything in the store
	// 8.95
	// 12.99
	// 8.99
	// 22.99
	// 399
	//
	// 5. the third book
	// map[author:Herman Melville category:fiction isbn:0-553-21311-3 price:8.99 title:Moby Dick]
	//
	// 6. the third book's author
	// Herman Melville
	//
	// 7. empty result: the third book does not have a "publisher" member
	//
	// 8. the last book in order
	// map[author:J. R. R. Tolkien category:fiction isbn:0-395-19395-8 price:22.99 title:The Lord of the Rings]
	//
	// 9. the first two books
	// map[author:Nigel Rees category:reference price:8.95 title:Sayings of the Century]
	// map[author:Evelyn Waugh category:fiction price:12.99 title:Sword of Honour]
	// map[author:Nigel Rees category:reference price:8.95 title:Sayings of the Century]
	// map[author:Evelyn Waugh category:fiction price:12.99 title:Sword of Honour]
	//
	// 10. all books with an ISBN number
	// map[author:Herman Melville category:fiction isbn:0-553-21311-3 price:8.99 title:Moby Dick]
	// map[author:J. R. R. Tolkien category:fiction isbn:0-395-19395-8 price:22.99 title:The Lord of the Rings]
	//
	// 11. all books cheaper than 10
	// map[author:Nigel Rees category:reference price:8.95 title:Sayings of the Century]
	// map[author:Herman Melville category:fiction isbn:0-553-21311-3 price:8.99 title:Moby Dick]
	//
	// 12. all member values and array elements contained in the input value
	// map[bicycle:{"color": "red", "price": 399} book:[
	//       {"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
	//       {"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
	//       {"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
	//       {"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99}
	//     ]]
	// [{"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95} {"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99} {"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99} {"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99}]
	// map[color:red price:399]
	// map[author:Nigel Rees category:reference price:8.95 title:Sayings of the Century]
	// map[author:Evelyn Waugh category:fiction price:12.99 title:Sword of Honour]
	// map[author:Herman Melville category:fiction isbn:0-553-21311-3 price:8.99 title:Moby Dick]
	// map[author:J. R. R. Tolkien category:fiction isbn:0-395-19395-8 price:22.99 title:The Lord of the Rings]
	// reference
	// Nigel Rees
	// Sayings of the Century
	// 8.95
	// fiction
	// Evelyn Waugh
	// Sword of Honour
	// 12.99
	// fiction
	// Herman Melville
	// Moby Dick
	// 0-553-21311-3
	// 8.99
	// fiction
	// J. R. R. Tolkien
	// The Lord of the Rings
	// 0-395-19395-8
	// 22.99
	// red
	// 399
}

func ExampleGetMany_functions() {
	// length() 函数示例 - 计算值的长度
	fmt.Println("1. length() - 计算数组元素的个数")
	r := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?length(@.category) == 9]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n2. length() - 筛选有 4 个成员的对象（有 isbn 字段的书）")
	r = jsonpath.GetMany(rfcExampleJSON, "$.store.book[?length(@) == 4]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	// count() 函数示例 - 计算节点列表中的节点数量
	fmt.Println("\n3. count() - 检查所有属性数量是否等于 2")
	r = jsonpath.GetMany(rfcExampleJSON, "$.store[?count(@.*) == 2]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	// value() 函数示例 - 将节点列表转换为值
	fmt.Println("\n4. value() - 获取 descendant nodes 中唯一值为 'red' 的节点")
	r = jsonpath.GetMany(rfcExampleJSON, "$.store[?value(@..color) == \"red\"]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	// 添加更多示例数据的 JSON
	functionsExampleJSON := `{
		"users": [
			{"name": "Bob", "email": "bob@example.com", "role": "admin"},
			{"name": "Alice", "email": "alice@example.com", "role": "user"},
			{"name": "Rob", "email": "rob@example.com", "role": "user"}
		],
		"products": [
			{"id": "A001", "name": "Apple", "category": "fruit"},
			{"id": "B002", "name": "Banana", "category": "fruit"},
			{"id": "C003", "name": "Carrot", "category": "vegetable"}
		],
		"dates": ["2024-01-15", "2024-02-20", "2024-03-25"]
	}`

	// match() 函数示例 - 完全匹配正则表达式
	fmt.Println("\n5. match() - 匹配以 2024-02 开头的日期")
	r = jsonpath.GetMany(functionsExampleJSON, "$.dates[?match(@, '2024-02-..')]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n6. match() - 匹配以 example.com 结尾的邮箱")
	r = jsonpath.GetMany(functionsExampleJSON, "$.users[?match(@.email, '.*@example.com')]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	// search() 函数示例 - 搜索包含匹配正则表达式的子串
	fmt.Println("\n7. search() - 名字包含 [BR]ob 的用户")
	r = jsonpath.GetMany(functionsExampleJSON, "$.users[?search(@.name, '[BR]ob')]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n8. search() - ID 包含数字 0 的产品")
	r = jsonpath.GetMany(functionsExampleJSON, "$.products[?search(@.id, '0')]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	fmt.Println("\n9. search() - 类别包含 'e' 的产品")
	r = jsonpath.GetMany(functionsExampleJSON, "$.products[?search(@.category, 'e')]")
	for _, v := range r {
		fmt.Println(v.Value())
	}

	// Output:
	// 1. length() - 计算数组元素的个数
	// map[author:Nigel Rees category:reference price:8.95 title:Sayings of the Century]
	//
	// 2. length() - 筛选有 4 个成员的对象（有 isbn 字段的书）
	// map[author:Nigel Rees category:reference price:8.95 title:Sayings of the Century]
	// map[author:Evelyn Waugh category:fiction price:12.99 title:Sword of Honour]
	//
	// 3. count() - 检查所有属性数量是否等于 2
	// map[color:red price:399]
	//
	// 4. value() - 获取 descendant nodes 中唯一值为 'red' 的节点
	// map[color:red price:399]
	//
	// 5. match() - 匹配以 2024-02 开头的日期
	// 2024-02-20
	//
	// 6. match() - 匹配以 example.com 结尾的邮箱
	// map[email:bob@example.com name:Bob role:admin]
	// map[email:alice@example.com name:Alice role:user]
	// map[email:rob@example.com name:Rob role:user]
	//
	// 7. search() - 名字包含 [BR]ob 的用户
	// map[email:bob@example.com name:Bob role:admin]
	// map[email:rob@example.com name:Rob role:user]
	//
	// 8. search() - ID 包含数字 0 的产品
	// map[category:fruit id:A001 name:Apple]
	// map[category:fruit id:B002 name:Banana]
	// map[category:vegetable id:C003 name:Carrot]
	//
	// 9. search() - 类别包含 'e' 的产品
	// map[category:vegetable id:C003 name:Carrot]
}
