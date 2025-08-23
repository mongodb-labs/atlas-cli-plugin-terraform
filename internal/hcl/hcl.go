package hcl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
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

// GetAttrExpr returns the expression of an attribute as a string.
func GetAttrExpr(attr *hclwrite.Attribute) string {
	if attr == nil {
		return ""
	}
	return strings.TrimSpace(string(attr.Expr().BuildTokens(nil).Bytes()))
}

// SetAttrInt sets an attribute to a number.
func SetAttrInt(body *hclwrite.Body, attrName string, number int) {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenNumberLit, Bytes: []byte(strconv.Itoa(number))},
	}
	body.SetAttributeRaw(attrName, tokens)
}

// GetAttrInt tries to get an attribute value as an int.
func GetAttrInt(attr *hclwrite.Attribute, errPrefix string) (int, error) {
	expr, diags := hclsyntax.ParseExpression(attr.Expr().BuildTokens(nil).Bytes(), "", hcl.InitialPos)
	if diags.HasErrors() {
		return 0, fmt.Errorf("%s: failed to parse number: %s", errPrefix, diags.Error())
	}
	val, diags := expr.Value(nil)
	if diags.HasErrors() {
		return 0, fmt.Errorf("%s: failed to evaluate number: %s", errPrefix, diags.Error())
	}
	if !val.Type().Equals(cty.Number) {
		return 0, fmt.Errorf("%s: attribute is not a number", errPrefix)
	}
	num, _ := val.AsBigFloat().Int64()
	return int(num), nil
}

// GetAttrString tries to get an attribute value as a string.
func GetAttrString(attr *hclwrite.Attribute) (string, error) {
	expr, diags := hclsyntax.ParseExpression(attr.Expr().BuildTokens(nil).Bytes(), "", hcl.InitialPos)
	if diags.HasErrors() {
		return "", fmt.Errorf("failed to parse string: %s", diags.Error())
	}
	val, diags := expr.Value(nil)
	if diags.HasErrors() {
		return "", fmt.Errorf("failed to evaluate string: %s", diags.Error())
	}
	if !val.Type().Equals(cty.String) {
		return "", fmt.Errorf("attribute is not a string")
	}
	return val.AsString(), nil
}

// TokensArray creates an array of objects.
func TokensArray(bodies []*hclwrite.Body) hclwrite.Tokens {
	tokens := make([]hclwrite.Tokens, 0)
	for i := range bodies {
		tokens = append(tokens, TokensObject(bodies[i]))
	}
	return EncloseBracketsNewLines(joinTokens(tokens...))
}

// TokensArraySingle creates an array of one object.
func TokensArraySingle(body *hclwrite.Body) hclwrite.Tokens {
	return TokensArray([]*hclwrite.Body{body})
}

// TokensObject creates an object.
func TokensObject(body *hclwrite.Body) hclwrite.Tokens {
	tokens := hclwrite.Tokens{tokenNewLine}
	tokens = append(tokens, RemoveLeadingNewline(body.BuildTokens(nil))...)
	return EncloseBraces(tokens, false)
}

// TokensFromExpr creates the tokens for an expression provided as a string.
func TokensFromExpr(expr string) hclwrite.Tokens {
	return hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte(expr)}}
}

// TokensFuncMerge creates the tokens for the HCL merge function.
func TokensFuncMerge(tokens ...hclwrite.Tokens) hclwrite.Tokens {
	params := EncloseNewLines(joinTokens(tokens...))
	ret := TokensFromExpr("merge")
	return append(ret, EncloseParens(params)...)
}

// TokensFuncConcat creates the tokens for the HCL concat function.
func TokensFuncConcat(tokens ...hclwrite.Tokens) hclwrite.Tokens {
	params := EncloseNewLines(joinTokens(tokens...))
	if len(tokens) == 1 {
		return tokens[0] // no need to concat if there's only one element
	}
	ret := TokensFromExpr("concat")
	return append(ret, EncloseParens(params)...)
}

// TokensFuncFlatten creates the tokens for the HCL flatten function.
func TokensFuncFlatten(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := TokensFromExpr("flatten")
	return append(ret, EncloseParens(EncloseBracketsNewLines(tokens))...)
}

// RemoveLeadingNewline removes the first newline if it exists to make the output prettier.
func RemoveLeadingNewline(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) > 0 && tokens[0].Type == hclsyntax.TokenNewline {
		return tokens[1:]
	}
	return tokens
}

// AppendComment adds a comment at the end of the body.
func AppendComment(body *hclwrite.Body, comment string) {
	body.AppendUnstructuredTokens(TokensComment(comment))
}

// TokensComment returns the tokens for a comment.
func TokensComment(comment string) hclwrite.Tokens {
	return hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenComment, Bytes: []byte("# " + comment + "\n")},
	}
}

// GetParser returns a parser for the given config and checks HCL syntax is valid
func GetParser(config []byte) (*hclwrite.File, error) {
	parser, diags := hclwrite.ParseConfig(config, "", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse Terraform config file: %s", diags.Error())
	}
	return parser, nil
}

// joinTokens joins multiple tokens with commas and newlines.
func joinTokens(tokens ...hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{}
	for i := range tokens {
		ret = append(ret, tokens[i]...)
		if i < len(tokens)-1 {
			ret = append(ret, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")}, tokenNewLine)
		}
	}
	return ret
}

// EncloseParens encloses tokens with parentheses, ( ).
func EncloseParens(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenOParen, Bytes: []byte("(")}}
	ret = append(ret, tokens...)
	return append(ret, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
}

// EncloseBraces encloses tokens with curly braces, { }.
func EncloseBraces(tokens hclwrite.Tokens, initialNewLine bool) hclwrite.Tokens {
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")}}
	if initialNewLine {
		ret = append(ret, tokenNewLine)
	}
	ret = append(ret, tokens...)
	return append(ret, &hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")})
}

// EncloseBrackets encloses tokens with square brackets, [ ].
func EncloseBrackets(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")}}
	ret = append(ret, tokens...)
	return append(ret, &hclwrite.Token{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")})
}

// EncloseNewLines encloses tokens with newlines at the beginning and end.
func EncloseNewLines(tokens hclwrite.Tokens) hclwrite.Tokens {
	ret := hclwrite.Tokens{tokenNewLine}
	ret = append(ret, tokens...)
	return append(ret, tokenNewLine)
}

// EncloseBracketsNewLines encloses tokens with square brackets and newlines, [ \n ... \n ].
func EncloseBracketsNewLines(tokens hclwrite.Tokens) hclwrite.Tokens {
	return EncloseBrackets(EncloseNewLines(tokens))
}

var tokenNewLine = &hclwrite.Token{Type: hclsyntax.TokenNewline, Bytes: []byte("\n")}
