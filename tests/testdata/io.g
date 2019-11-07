Program <- Sep? Expr ( Sep Expr )* Sep?
# TODO: the assignments are handled in the grammar, but should be tree-rewritten.
Statement <- Ident Assign Expr / Expr
Expr <- Message+
Message <- Ident Args / Ident / Literal
# TODO: the assigments are not allowed in the arguments now.
Args <- _ "(" NL* Expr ( _ "," NL* Expr )* _ ")"

Ident <- _ < [A-Za-z_][A-Za-z0-9_]* >
Literal <- Number / String
Number <- _ < [0-9]+ >
String <- _ '"' < ( '\"' / !'"'  . )* > '"'
Assign <- _ < (":=" / "=") >

Sep <- NL+ / _ ";" NL*
NL <-  _ ( "#" ( ![\n\r] . )* )? [\n\r]+
_ <- [ \t]*
