# The MONKEY D. Programming Language Compiler

### Flow thorugh Modules

| 1. Source Code | -> | 2. LEXER  | -> | 3. PARSER | -> | 4. COMPILER | -> | 5. VIRTUAL MACHINE |

___

### Data Structures

| 1. String | -> | 2. Tokens | -> | 3. Abstract Syntax Tree | -> | 4. Bytecode | -> | 5. Objects |

___

## Major Parts

- the lexer
- the parser
- the Abstract Syntax Tree (AST)
- the internal object system
- the evaluator
- the compuler
- the virtual machine


#### Opcode - operation Code

- one byte wide
- has arbitrary but unique value
- is the first byte in the instruction


## Virtual Machine is based on the Stack (stack machine)



## Features
- C-like syntax
- variable bindings
    ```
        // string
        let name = "Ioan";

        // int
        let age = 23;    
        let total = (33 * (10 * 10)) + 3;
        
        // array
        let pets = ["Barni", "Mandarin", "Dora", "Rosa"];
        pets[1] // "Mandarin"

        // hash 
        let ioan = {"name": "Ioan", "age": 23};
        ioan["age"] // 23
    ```
- integers and booleans

- arithmetic expressions
- built-in functions
    ```
        // bind functions to names
        let double = fn(a) { return a * 2 };
        add(2) // 4

        // recursive call
        let fibinacci = fn(x) {
            if (x == 0) {
                0
            } else {
                if (x == 1) {
                    1
                } else {
                    fibonacci(x - 1) + fibonacci(x - 2);
                }
            }
        };
    ```
- first-class and higher-order functions - functions that take other functions as arguments
    ```
        let twice = fn(f, x) {
            return f(f(x));
        };

        let addTwo = fn(x) {
            return x + 2
        };

        twice(addTwo, 2); // 6
    ```
- closures
- a string data structure
- an array data structure
- a hash data structure



## Parsing

- From root node to childern

- "Top-down operator precedence" - recursive descent parser - "Pratt parser"

 - [BNF for EcmaScript.jj](https://tomcopeland.blogs.com/EcmaScript.html#prod7)


## Expressions In MONKEY D.

- Everything besides `let` and `return` statements is an expression

    - Prefix Operators
        - `-3`
        - `!true`
        - `!false`

    - Infix operators or "binary operators"
        - `3 + 3`
        - `3 - 3`
        - `3 / 3`
        - `3 * 3`
    
    - Comparison Operators
        - `foo == bar`
        - `foo != bar`
        - `foo < bar`
        - `foo > bar`

    - Parentheses to group expressions and influence the order of evaluation
        - `3 * (3 + 3)`
        - `((3 + 3) * 3) * 3`
    
    - Call expressions
        - `add(2, 3)`
        - `add(add(2, 3), add(3, 10))`
        - `max(3, add(3, (3 * 3)))`

    - Identifiers expressions
        - `foo * bar / foobar`
        - `add(foo, bar)`

    - Function literals are expressions
        - `let add = fn(x, y) { return x + y };`
    
    - Function literal in place of an identifier
        - `fn(x, y) { return x + y }(5, 5);`
        - `(fn(x) { return x }(5) + 10) * 10;`
    
    - if expression
        - `let result = if (10 > 5) { true } else { false };`
        
            `result // => true`

    - Using function literal as an argument when calling another function
        - `aFunc(x, y, fn(x, y) { return x > y; });`

# REPL
 - Read
 - Evaluate
 - Print
 - Loop

    ```
    go run main.go
    Hello <user>! This is the Monkey D.programming language!
    Feel free to type in commands
    >> let a = 3 * 3 * 3;
    let a = ((3 * 3) * 3);
    >> x * y / 3 - 33 * 2 - 13
    ((((x * y) / 3) - (33 * 2)) - 13)
    >> false == true
    (false == true)
    ```

## Test Driving Arrays

### Map
    let map = fn(arr, f) {
        let iter = fn(arr, accumulated) {
            if (len(arr) == 0) {
                accumulated
            } else {
                iter(rest(arr), push(accumulated, f(first(arr))));
            }
        };

        iter(arr, []);
    };

    
    >> let a = [1, 2, 3, 4];
    >> let double = fn(x) { x * 2 };
    >> map(a, double);
    [2, 4, 6, 8]
    
### Reduce

    let reduce = fn(arr, initial, f) {
        let iter = fn(arr, result) {
            if (len(arr) == 0) {
                result
            } else {
                iter(rest(arr), f(result, first(arr)));
            }
        };
        iter(arr, initial);
    };


    let sum = fn(arr) {
        reduce(arr, 0, fn(initial, el) { initial + el });
    };


    >> sum([1, 2, 3, 4, 5]);
    15


## Hashes

    >> {"name": "Monkey D.", "age": -1, "type": "Language", "status": "distinguished"}
    {age: -1, type: Language, status: distinguished, name: Monkey D.}
    >> let ioan = {"name": "Ioan", "age": 33};



    >> let users = [{"name": "Glen", "age": 34}, {"name": "Peter", "age": 29}];
    >> users[0]["name"];
    Glen
    >> users[1]["name"];
    Peter
    >> users[1]["age"] + users[0]["age"];
    63
    >> let getName = fn(user) { user["name"]; };
    >> getName(users[0]);
    Glen
    >> getName(users[1]);
    Peter


## Builtins:

### len

    >> len("abc")
    3
    >> len(["a", "b", "c"])
    3

### first

    >> first(["a", "b", "c"])
    a

### last

    >> last(["a", "b", "c"])
    c

### rest

    >> rest(["a", "b", "c"])
    [b, c]

### push

    >> push([], "a")
    [a]

### puts

    >> puts("Hello Monkey D!");   
    Hello Monkey D!
    null
