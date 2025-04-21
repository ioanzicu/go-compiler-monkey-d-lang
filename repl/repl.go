package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/ioanzicu/monkeyd/compiler"
	"github.com/ioanzicu/monkeyd/lexer"
	"github.com/ioanzicu/monkeyd/object"
	"github.com/ioanzicu/monkeyd/parser"
	"github.com/ioanzicu/monkeyd/vm"
)

// R - READ
// E - EVALUATE
// P - PRINT
// L - LOOP

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalsSize)

	symbolTable := compiler.NewSymbolTable()
	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	// Read input from Terminal
	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		if line == "exit" {
			fmt.Println("Bye, bye ...!")
			return
		}

		// Pass input to the lexer
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		comp := compiler.NewWithState(symbolTable, constants)
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "Woops! Compilation failed:\n %s\n", err)
			continue
		}

		code := comp.Bytecode()
		constants = code.Constants

		machine := vm.NewWithGlobalsStore(code, globals)
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "Woops! Executing bytecode failed:\n %s\n", err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		io.WriteString(out, lastPopped.Inspect())
		io.WriteString(out, "\n")
	}
}

const MONKEY_FACE = `
      .--.  .-"__,__"-. .--.
     / .. \/  .-. .-.  \/ .. \
    | |  '|  /   Y   \  |'  | |
    | \   \  \ 0 | 0 /  /   / |
     \ '-, \.-"""""""-./,-'  /
      ''-' /_   ^ ^   _\ '-''
          | \._    _./  |
          \  \  '~' /   /
           '._ '-=-' _.'
              '-----'
`

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, MONKEY_FACE)
	io.WriteString(out, "Woops! We ran into some monkey business here!\n")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
