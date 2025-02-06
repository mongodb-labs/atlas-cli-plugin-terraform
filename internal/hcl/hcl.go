package hcl

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// MoveAttr deletes an attribute from fromBody and adds it to toBody.
func MoveAttr(fromBody, toBody *hclwrite.Body, fromAttrName, toAttrName, errPrefix string) error {
	tokens, err := PopAttr(fromBody, fromAttrName, errPrefix)
	if err == nil {
		toBody.SetAttributeRaw(toAttrName, tokens)
	}
	return err
}

// PopAttr deletes an attribute and returns it value.
func PopAttr(body *hclwrite.Body, attrName, errPrefix string) (hclwrite.Tokens, error) {
	attr := body.GetAttribute(attrName)
	if attr == nil {
		return nil, fmt.Errorf("%s: attribute %s not found", errPrefix, attrName)
	}
	tokens := attr.Expr().BuildTokens(nil)
	body.RemoveAttribute(attrName)
	return tokens, nil
}

// SetAttrInt sets an attribute to a number.
func SetAttrInt(body *hclwrite.Body, attrName string, number int) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenNumberLit, Bytes: []byte(strconv.Itoa(number))},
	}
	body.SetAttributeRaw(attrName, tokens)
}

// TokensArrayObject creates an array with a single object.
func TokensArrayObject(file *hclwrite.File) hclwrite.Tokens {
	ret := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
	}
	ret = append(ret, TokensObject(file)...)
	ret = append(ret,
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
	return ret
}

// TokensObject creates an object.
func TokensObject(file *hclwrite.File) hclwrite.Tokens {
	ret := hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("{")},
		{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")},
	}
	ret = append(ret, file.BuildTokens(nil)...)
	ret = append(ret,
		&hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("}")})
	return ret
}

// AppendComment adds a comment at the end of the body.
func AppendComment(body *hclwrite.Body, comment string) {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte("# " + comment + "\n")},
	}
	body.AppendUnstructuredTokens(tokens)
}

// GetParser returns a parser for the given config and checks HCL syntax is valid
func GetParser(config []byte) (*hclwrite.File, error) {
	parser, diags := hclwrite.ParseConfig(config, "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse Terraform config file: %s", diags.Error())
	}
	return parser, nil
}
