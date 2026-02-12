package jsonpath_test

import (
	"fmt"

	"github.com/saltfishpr/jsonpath"
)

// RFC 9535 示例 JSON (Figure 1)
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

func Example_basic() {
	// 基础查询：获取 store 中所有书籍的作者
	authors := jsonpath.GetMany(rfcExampleJSON, "$.store.book[*].author")
	for i, author := range authors {
		fmt.Printf("作者 %d: %s\n", i+1, author.String())
	}

	// 单结果查询：获取第三本书的作者
	author := jsonpath.Get(rfcExampleJSON, "$..book[2].author")
	if author.Exists() {
		fmt.Printf("第三本书的作者: %s\n", author.String())
	}

	// Output:
	// 作者 1: Nigel Rees
	// 作者 2: Evelyn Waugh
	// 作者 3: Herman Melville
	// 作者 4: J. R. R. Tolkien
	// 第三本书的作者: Herman Melville
}

func Example_wildcard() {
	// 通配符：获取 store 中的所有成员
	items := jsonpath.GetMany(rfcExampleJSON, "$.store.*")
	fmt.Printf("store 中有 %d 个成员\n", len(items))

	// 通配符：获取所有价格（包括书和自行车）
	prices := jsonpath.GetMany(rfcExampleJSON, "$.store..price")
	for _, price := range prices {
		fmt.Printf("价格: %s\n", price.String())
	}

	// Output:
	// store 中有 2 个成员
	// 价格: 8.95
	// 价格: 12.99
	// 价格: 8.99
	// 价格: 22.99
	// 价格: 399
}

func Example_descendant() {
	// 后代段：获取所有 price 字段（使用 ..）
	prices := jsonpath.GetMany(rfcExampleJSON, "$..price")
	for _, price := range prices {
		fmt.Printf("价格: %s\n", price.String())
	}

	// 后代段：获取所有 category 字段
	categories := jsonpath.GetMany(rfcExampleJSON, "$..category")
	for _, cat := range categories {
		fmt.Printf("分类: %s\n", cat.String())
	}

	// Output:
	// 价格: 8.95
	// 价格: 12.99
	// 价格: 8.99
	// 价格: 22.99
	// 价格: 399
	// 分类: reference
	// 分类: fiction
	// 分类: fiction
	// 分类: fiction
}

func Example_index() {
	// 索引选择器：获取第一本书
	firstBook := jsonpath.Get(rfcExampleJSON, "$.store.book[0]")
	fmt.Printf("第一本书: %s\n", firstBook.Get("$.title").String())

	// 索引选择器：获取最后一本书（负索引）
	lastBook := jsonpath.Get(rfcExampleJSON, "$.store.book[-1]")
	fmt.Printf("最后一本书: %s\n", lastBook.Get("$.title").String())

	// 多索引选择器：获取第一本和第三本书
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[0,2]")
	for _, book := range books {
		fmt.Printf("书籍: %s\n", book.Get("$.title").String())
	}

	// Output:
	// 第一本书: Sayings of the Century
	// 最后一本书: The Lord of the Rings
	// 书籍: Sayings of the Century
	// 书籍: Moby Dick
}

func Example_slice() {
	// 切片选择器：获取前两本书
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[0:2]")
	for _, book := range books {
		fmt.Printf("书籍: %s\n", book.Get("$.title").String())
	}

	// 切片选择器：从第2本到最后一本
	books2 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[1:]")
	fmt.Printf("从第2本开始共 %d 本\n", len(books2))

	// 切片选择器：每隔一本选取（步长为2）
	books3 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[::2]")
	for _, book := range books3 {
		fmt.Printf("隔本选取: %s\n", book.Get("$.title").String())
	}

	// 切片选择器：反向切片
	books4 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[::-1]")
	fmt.Printf("第一本(倒序): %s\n", books4[0].Get("$.title").String())

	// Output:
	// 书籍: Sayings of the Century
	// 书籍: Sword of Honour
	// 从第2本开始共 3 本
	// 隔本选取: Sayings of the Century
	// 隔本选取: Moby Dick
	// 第一本(倒序): The Lord of the Rings
}

func Example_filter_comparison() {
	// 过滤器：价格小于10的书
	cheapBooks := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?@.price < 10]")
	fmt.Printf("廉价书数量: %d\n", len(cheapBooks))
	for _, book := range cheapBooks {
		fmt.Printf("廉价书: %s (%s)\n", book.Get("$.title").String(), book.Get("$.price").String())
	}

	// 过滤器：分类为 fiction 的书
	fictionBooks := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?@.category == 'fiction']")
	fmt.Printf("小说数量: %d\n", len(fictionBooks))

	// 过滤器：有 isbn 字段的书
	booksWithISBN := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?@.isbn]")
	fmt.Printf("有ISBN的书数量: %d\n", len(booksWithISBN))

	// Output:
	// 廉价书数量: 2
	// 廉价书: Sayings of the Century (8.95)
	// 廉价书: Moby Dick (8.99)
	// 小说数量: 3
	// 有ISBN的书数量: 2
}

func Example_filter_logical() {
	// 逻辑与：价格大于10且分类为 fiction
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?(@.price > 10 && @.category == 'fiction')]")
	for _, book := range books {
		fmt.Printf("昂贵小说: %s\n", book.Get("$.title").String())
	}

	// 逻辑或：价格小于9或价格大于20
	books2 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?(@.price < 9 || @.price > 20)]")
	fmt.Printf("价格<9或>20: %d 本\n", len(books2))

	// 逻辑非：分类不为 fiction
	books3 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?!(@.category == 'fiction')]")
	for _, book := range books3 {
		fmt.Printf("非小说: %s\n", book.Get("$.category").String())
	}

	// Output:
	// 昂贵小说: Sword of Honour
	// 昂贵小说: The Lord of the Rings
	// 价格<9或>20: 3 本
	// 非小说: reference
}

func Example_function_length() {
	// length() 函数：在过滤器中使用
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?length(@.title) > 15]")
	for _, book := range books {
		fmt.Printf("长标题: %s\n", book.Get("$.title").String())
	}

	// Output:
	// 长标题: Sayings of the Century
	// 长标题: The Lord of the Rings
}

func Example_function_match() {
	// match() 函数：完整匹配正则表达式
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?match(@.category, 'fict.*')]")
	fmt.Printf("匹配 'fict.*': %d 本\n", len(books))

	// match() 函数：精确匹配
	books2 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?match(@.category, 'fiction')]")
	fmt.Printf("精确匹配 'fiction': %d 本\n", len(books2))

	// match() 函数：不匹配的情况
	books3 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?match(@.category, 'ref.*')]")
	fmt.Printf("匹配 'ref.*': %d 本\n", len(books3))

	// Output:
	// 匹配 'fict.*': 3 本
	// 精确匹配 'fiction': 3 本
	// 匹配 'ref.*': 1 本
}

func Example_function_search() {
	// search() 函数：搜索包含子字符串的值
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?search(@.title, 'of')]")
	for _, book := range books {
		fmt.Printf("包含 'of': %s\n", book.Get("$.title").String())
	}

	// search() 函数：使用正则表达式
	books2 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?search(@.author, '.*Melville')]")
	for _, book := range books2 {
		fmt.Printf("作者匹配: %s\n", book.Get("$.author").String())
	}

	// Output:
	// 包含 'of': Sayings of the Century
	// 包含 'of': Sword of Honour
	// 包含 'of': The Lord of the Rings
	// 作者匹配: Herman Melville
}

func Example_complex_filter() {
	// 复杂过滤器：组合多个条件
	// 找出价格在10到20之间，且分类为 fiction 的书
	books := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?(@.price >= 10 && @.price <= 20 && @.category == 'fiction')]")
	for _, book := range books {
		fmt.Printf("符合条件的书: %s, 价格: %s\n",
			book.Get("$.title").String(), book.Get("$.price").String())
	}

	// 复杂过滤器：使用括号改变优先级
	books2 := jsonpath.GetMany(rfcExampleJSON, "$.store.book[?((@.price < 10 || @.price > 20) && @.category == 'fiction')]")
	fmt.Printf("(价格<10或>20)且是fiction: %d 本\n", len(books2))

	// Output:
	// 符合条件的书: Sword of Honour, 价格: 12.99
	// (价格<10或>20)且是fiction: 2 本
}

func Example_special_member_names() {
	// 使用括号表示法访问特殊字段名
	jsonData := `{"field with spaces": "value1", "field-with-dash": "value2"}`

	// 使用字符串形式访问带空格的字段名
	result1 := jsonpath.Get(jsonData, `$["field with spaces"]`)
	fmt.Printf("带空格的字段: %s\n", result1.String())

	result2 := jsonpath.Get(jsonData, `$["field-with-dash"]`)
	fmt.Printf("带连字符的字段: %s\n", result2.String())

	// Output:
	// 带空格的字段: value1
	// 带连字符的字段: value2
}

func Example_root_vs_current() {
	// 根节点 $ vs 当前节点 @
	jsonData := `{
		"store": {
			"book": [{"price": 10}, {"price": 20}],
			"bicycle": {"price": 50}
		}
	}`

	// 使用 @ 访问当前节点
	books := jsonpath.GetMany(jsonData, "$.store.book[?@.price > 15]")
	fmt.Printf("价格>15的书: %d 本\n", len(books))

	// 使用 $ 访问根节点
	books2 := jsonpath.GetMany(jsonData, "$.store.book[?@.price > $.store.bicycle.price]")
	fmt.Printf("价格>自行车的书: %d 本\n", len(books2))

	// Output:
	// 价格>15的书: 1 本
	// 价格>自行车的书: 0 本
}

func Example_array_traversal() {
	// 数组遍历的各种方式
	jsonData := `{
		"items": [
			{"name": "a", "value": 1},
			{"name": "b", "value": 2},
			{"name": "c", "value": 3}
		]
	}`

	// 使用通配符获取所有元素
	all := jsonpath.GetMany(jsonData, "$.items[*]")
	fmt.Printf("所有元素: %d 个\n", len(all))

	// 使用过滤器选择特定元素
	filtered := jsonpath.GetMany(jsonData, "$.items[?@.value > 1]")
	fmt.Printf("value>1的元素: %d 个\n", len(filtered))

	// 组合多个索引
	selected := jsonpath.GetMany(jsonData, "$.items[0,2]")
	fmt.Printf("索引0和2: %d 个\n", len(selected))

	// Output:
	// 所有元素: 3 个
	// value>1的元素: 2 个
	// 索引0和2: 2 个
}

func Example_existence_test() {
	// 存在性测试：检查字段是否存在
	jsonData := `{
		"users": [
			{"name": "Alice", "age": 30},
			{"name": "Bob"},
			{"name": "Charlie", "age": 25}
		]
	}`

	// 使用测试表达式找出有 age 字段的用户
	usersWithAge := jsonpath.GetMany(jsonData, "$.users[?@.age]")
	fmt.Printf("有age字段: %d 个\n", len(usersWithAge))

	// 使用逻辑非找出没有 age 字段的用户
	usersWithoutAge := jsonpath.GetMany(jsonData, "$.users[?!@.age]")
	for _, user := range usersWithoutAge {
		fmt.Printf("无age字段: %s\n", user.Get("$.name").String())
	}

	// Output:
	// 有age字段: 2 个
	// 无age字段: Bob
}
