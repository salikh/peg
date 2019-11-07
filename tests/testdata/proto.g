# top rule
Source <- _ SyntaxDecl PackageDecl? MessageDecl*

# grammar rules
SyntaxDecl <- "syntax" _ "=" _ QuotedLiteral ";" _
PackageDecl <- "package" _ Identifier _ ";" _
MessageDecl <- "message" _ Identifier _ "{" _ FieldDecl * "}" _
FieldDecl <- FieldSpec _ Type _ Identifier _ "=" _ Integer _ ";" _
FieldSpec <- "optional" / "repeated" / "required"
Type <- "int32" / "int64" / "bool" / "float" / "double" / "string"

# tokens
QuotedLiteral <- '"' < ( '\"' / !'"' . ) *  > '"'
Identifier <- [a-zA-Z_][a-zA-Z_0-9]*
Integer <- [0-9]+

# spacing
_ <- ([\n\t ] / "//" ( !"\n" . )* "\n")*
