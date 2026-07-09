// YARA rules for SECRET detection
// Covers AWS keys, private key PEM blocks, GitHub tokens, generic API keys, Bearer JWTs

rule AWS_Access_Key
{
    meta:
        category = "SECRET"
        description = "AWS Access Key ID"
        severity = "critical"
    strings:
        $aws_key = /AKIA[0-9A-Z]{16}/ ascii wide
    condition:
        $aws_key
}

rule RSA_Private_Key
{
    meta:
        category = "SECRET"
        description = "PEM-encoded RSA private key block"
        severity = "critical"
    strings:
        $begin_rsa = "-----BEGIN RSA PRIVATE KEY-----" ascii wide
    condition:
        $begin_rsa
}

rule EC_Private_Key
{
    meta:
        category = "SECRET"
        description = "PEM-encoded EC private key block"
        severity = "critical"
    strings:
        $begin_ec = "-----BEGIN EC PRIVATE KEY-----" ascii wide
    condition:
        $begin_ec
}

rule Generic_Private_Key
{
    meta:
        category = "SECRET"
        description = "PEM-encoded generic private key block"
        severity = "critical"
    strings:
        $begin_pk = "-----BEGIN PRIVATE KEY-----" ascii wide
    condition:
        $begin_pk
}

rule GitHub_Token
{
    meta:
        category = "SECRET"
        description = "GitHub Personal Access Token (classic ghp_ prefix)"
        severity = "high"
    strings:
        $ghp = /ghp_[0-9A-Za-z]{36}/ ascii wide
    condition:
        $ghp
}

rule API_Key_Assignment
{
    meta:
        category = "SECRET"
        description = "Generic api_key assignment pattern"
        severity = "medium"
    strings:
        $api_key1 = "api_key=" nocase ascii wide
        $api_key2 = "api-key=" nocase ascii wide
        $api_key3 = "apikey=" nocase ascii wide
    condition:
        any of them
}

rule Bearer_Token
{
    meta:
        category = "SECRET"
        description = "Bearer token in Authorization header"
        severity = "high"
    strings:
        $bearer = "Bearer " ascii wide nocase
    condition:
        $bearer
}
