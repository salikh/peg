# PEG, yet another parsing expression grammar for Go

Why another implementation for PEGs? This implementation was initially created
in 2014 when there were not as many PEG implementations for Go available as
there are now, and here are the reasons why it still may be useful to have this
implementation:

*   The same grammar definition can be used both with dynamic runtime
    implementation of parser as well as with parser generator. Both runtime
    dynamic parser and parser generator produce the syntax tree in the same
    format.

*   The parser generates a syntax tree that can immediately be inspected or
    manipulated without needing to write semantic actions. There is a library
    for serializing and parsing the syntax trees. The parsers uses a few
    conventions to ignore unimportant elements (e.g. whitespace) so that the
    syntactic tree may be quite useful out of the box.

*   The converter from syntax tree to semantic tree can be written in an
    imperative style that is easier to maintain than semantic actions embedded
    directly in the grammar.

*   This parser supports backwards parsing.

*   Last, but not the least, all my parsing-related projects use this parser,
    which makes it particularly valuable for me personally.

## Grammar definition rules

A parsing expression grammar consists of the rules in the following format:

    Name <- RHS

By convention, each rule starts on its own line, although it is okay to split
the right-hand side of the rule to multiple lines without using any line
continuation markers.

The elements that are allowed on the right hand side are mostly standard for
PEGs:

*   Sequence: `A B C`. The elements of sequence must match in sequence.
*   Ordered choice: `A / B / C`. The first match that is successful wins, and
    the subsequent choices are not considered.
*   Grouping: `( A B C )`. Groups allow to apply repetition or predicate
    operators to more than one match rule.
*   Repetition: `A+ B* C?`. `+` means 1 or more matches, `*` means 0 or more
    matches, and `?` means 0 or 1 matches.
*   Predicates and negative predicates: `&A !B`. `&` matches if the next term
    matches, but does not consume any of the input. `!` mathes if the next term
    does not match, and also does not consume any input.
*   Literals and character classes: `"abc" 'xyz' [012]`. The literals and
    character classes accept the escape sequences that are compatible with
    `strconv.Unquote`. The following Unicode character classes are also
    recognized: `[:alpha:]`, `[:digit:]`, `[:space:]`, `[:lower:]`, `[:upper:]`,
    `[:punct:]`, `[:print:]`, `[:graph:]`, `[:cntrl:]`, `[:alnum:]`, `[:any:]`.
*   Wildcard character match: `.`
*   String capture: `< A >`. This may be the only non-standard element of the
    parser grammar. If a rule defines a string capture, the part of the input
    that has matched is stored into the `Node.Text` field. By convention, if a
    rule does not have any node children or text captures, it is not stored into
    the syntax tree, so specifying a capture is a way to force a rule to always
    generate a node in the syntactic tree. See also
    `parser2.ParserOptions.SkipEmptyNodes`.

## Running the dynamic parser

Here is a snippet of code on how to invoke a dynamic parser (with error handling
elided):

    var grammarSource = `Source <- A* B*
    A <- "a"* _
    B <- "b"* _
    _ <- [ \t\n]*`

    grammar, err := parser.New(grammarSource)
    source := "aaabbb"
    result, err := grammar.Parse(source)
    result.ComputeContent()  // Optional.
    fmt.Printf("Parse tree:\n%s\n", result.Tree)

The result of the parsing is a syntactic parse tree that is returned in the
`Result.Tree` field and has the type `*parser.Node`. The most important fileds
in `parser.Node` are:

    type Node struct {
      // Label determines the type of the node, usually corresponding
      // to the rule name (LHS of the parser rule).
      Label string
      // Text is a captured text, if a rule defines a capture region.
      Text string
      // The children of this node.
      Children []*Node
      // The byte position of the first character consumed by this node.
      Pos int
      // The number of bytes consumed by this Node and its children.
      Len int
      // ... some fields are omitted.
    }

`parser.Node` also contains a few fields that are optionally computed by calling
`result.ComputeContent()` and that can be handy when providing user feedback
about locations in the parsed source, or when edits are applied to the syntax
tree and parsed content needs to be reconstructed:

    type Node struct {
      // ... some fields are omitted.

      // Pieces of parsed source that was consumed by this node,
      // interleaved with content belonging to Children.
      Content []string
      // The line number of the first character consumed by this node. 1-based.
      Row int
      // The column number of the first character consumed by this node. 0-based.
      Col int
    }

The syntactic parse trees can be pretty-printed and parsed back using the code
in `tree/` subpackage.

## How to develop and test the parser and parser generator.

Note: this project currently only supports Linux and Unix derivatives (e.g.
MacOS X).

Since this project contains both dynamic parser and static parser generator,
some effort is required to keep those in sync. This is facilitated by
maintaining the parser tests in a special format that is shared between test
suites for dynamic and generated parsers.

The source of the tests is kept in the subdirectory `tests/`. Since the tests
for parser generator are generated, one first needs to run the generator on
tests:

    go generate ./...
    go test ./...

# License

Apache-2.0; see LICENSE for details.

# Disclaimer

This project is not an official Google project. It is not supported by Google
and Google specifically disclaims all warranties as to its quality,
merchantability, or fitness for a particular purpose.
