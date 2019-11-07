Grammar <- Rule+ _

Rule <- _ Ident _ '<' '-' RHS EndOfLine? 
RHS <- Terms ( _ '/' Terms ) *
Terms <- Term+
Term <- Parens / NegPred / Pred / Capture / CharClass / Literal / Ident / Special
Special <- _ < [*?.+] >
Parens <- _ '(' RHS _ ')'
NegPred <- _ '!' Term 
Pred <- _ '&' Term 
Capture <- _ '<' RHS _ '>'

Literal <- _ '"' < ( !'"' . ) * > '"' / _ "'" < ( !"'" . )* > "'"
Ident <- [ \t]* < [a-zA-Z_][a-zA-Z0-9_]* >
CharClass <- _ '[' < ( !']' . ) * > ']'

EndOfLine <- [ \t]* ( "\r\n" / "\r" / "\n")
_ <- ( [ \t\r\n] / '#' ( !"\n" .)* "\n" )*
