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
