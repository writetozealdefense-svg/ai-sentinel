// YARA rules for SOURCE CODE detection
// Detects Python, JavaScript, C/C++, Java, Go, Ruby snippets embedded in content

rule Python_Source_Code
{
    meta:
        category = "CODE"
        description = "Python source code snippet"
        severity = "low"
    strings:
        $def    = /\bdef [a-zA-Z_][a-zA-Z0-9_]*\(/ ascii wide
        $import = /\bimport [a-zA-Z_]/ ascii wide
        $from   = /\bfrom [a-zA-Z_][\w.]* import\b/ ascii wide
        $class  = /\bclass [A-Z][a-zA-Z0-9_]*/ ascii wide
    condition:
        2 of them
}

rule JavaScript_Source_Code
{
    meta:
        category = "CODE"
        description = "JavaScript/TypeScript source code snippet"
        severity = "low"
    strings:
        $function = /\bfunction [a-zA-Z_][a-zA-Z0-9_]*\(/ ascii wide
        $const    = /\bconst [a-zA-Z_]/ ascii wide
        $let      = /\blet [a-zA-Z_]/ ascii wide
        $require  = /\brequire\(/ ascii wide
    condition:
        2 of them
}

rule C_CPP_Source_Code
{
    meta:
        category = "CODE"
        description = "C or C++ source code snippet"
        severity = "low"
    strings:
        $include = /#include [<"]/ ascii wide
        $main    = /\bint main\(/ ascii wide
        $printf  = /\bprintf\(/ ascii wide
    condition:
        $include or ($main and $printf)
}

rule Java_Source_Code
{
    meta:
        category = "CODE"
        description = "Java source code snippet"
        severity = "low"
    strings:
        $pub_cls = /\bpublic class [A-Z]/ ascii wide
        $pkg     = /\bpackage [a-z][a-z0-9_.]*;/ ascii wide
        $import  = /\bimport [a-z][a-z0-9_.]*\.[A-Z]/ ascii wide
        $void    = /\bpublic (static )?void / ascii wide
    condition:
        2 of them
}

rule SQL_Query
{
    meta:
        category = "CODE"
        description = "SQL query embedded in content"
        severity = "low"
    strings:
        $select = "SELECT" nocase fullword
        $from   = "FROM" nocase fullword
        $insert = "INSERT INTO" nocase
        $update = "UPDATE" nocase fullword
        $set_kw = "SET" nocase fullword
        $drop   = "DROP TABLE" nocase
    condition:
        ($select and $from) or $insert or ($update and $set_kw) or $drop
}
