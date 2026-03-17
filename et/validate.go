package et

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"

	"github.com/cgalvisleon/et/msg"
)

var rePhone = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

type Rule interface {
	Validate(Json) error
}

type StringRule struct {
	name     string
	notEmpty bool
}

/**
* Str
* @param name string
* @return *StringRule
**/
func Str(name string) *StringRule {
	return &StringRule{name: name}
}

/**
* NotEmpty
* @return *StringRule
**/
func (r *StringRule) NotEmpty() *StringRule {
	r.notEmpty = true
	return r
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *StringRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	str, ok := v.(string)
	if !ok {
		return fmt.Errorf(msg.MSG_STRING_REQUIRED, r.name)
	}

	if r.notEmpty && str == "" {
		return fmt.Errorf(msg.MSG_STRING_NOT_EMPTY, r.name)
	}

	return nil
}

type IntRule struct {
	name string
	min  *int
	max  *int
}

func Int(name string) *IntRule {
	return &IntRule{name: name}
}

/**
* Min
* @param v int
* @return *IntRule
**/
func (r *IntRule) Min(v int) *IntRule {
	r.min = &v
	return r
}

/**
* Max
* @param v int
* @return *IntRule
**/
func (r *IntRule) Max(v int) *IntRule {
	r.max = &v
	return r
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *IntRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	var num int

	switch t := v.(type) {
	case int:
		num = t
	case float64:
		num = int(t)
	default:
		return fmt.Errorf(msg.MSG_INT_REQUIRED, r.name)
	}

	if r.min != nil && num < *r.min {
		return fmt.Errorf(msg.MSG_INT_MIN, r.name, *r.min)
	}

	if r.max != nil && num > *r.max {
		return fmt.Errorf(msg.MSG_INT_MAX, r.name, *r.max)
	}

	return nil
}

type FloatRule struct {
	name string
	min  *float64
	max  *float64
}

func Float(name string) *FloatRule {
	return &FloatRule{name: name}
}

/**
* Min
* @param v float64
* @return *FloatRule
**/
func (r *FloatRule) Min(v float64) *FloatRule {
	r.min = &v
	return r
}

/**
* Max
* @param v float64
* @return *FloatRule
**/
func (r *FloatRule) Max(v float64) *FloatRule {
	r.max = &v
	return r
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *FloatRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	num, ok := v.(float64)
	if !ok {
		return fmt.Errorf(msg.MSG_FLOAT_REQUIRED, r.name)
	}

	if r.min != nil && num < *r.min {
		return fmt.Errorf(msg.MSG_FLOAT_MIN, r.name, *r.min)
	}

	if r.max != nil && num > *r.max {
		return fmt.Errorf(msg.MSG_FLOAT_MAX, r.name, *r.max)
	}

	return nil
}

type ArrayRule struct {
	name     string
	notEmpty bool
}

func Array(name string) *ArrayRule {
	return &ArrayRule{name: name}
}

/**
* NotEmpty
* @return *ArrayRule
**/
func (r *ArrayRule) NotEmpty() *ArrayRule {
	r.notEmpty = true
	return r
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *ArrayRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	arr, ok := v.([]interface{})
	if !ok {
		return fmt.Errorf(msg.MSG_ARRAY_REQUIRED, r.name)
	}

	if r.notEmpty && len(arr) == 0 {
		return fmt.Errorf(msg.MSG_ARRAY_NOT_EMPTY, r.name)
	}

	return nil
}

type EmailRule struct {
	name string
}

func Email(name string) *EmailRule {
	return &EmailRule{name: name}
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *EmailRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	str, ok := v.(string)
	if !ok {
		return fmt.Errorf(msg.MSG_STRING_REQUIRED, r.name)
	}

	_, err := mail.ParseAddress(str)
	if err != nil {
		return fmt.Errorf(msg.MSG_EMAIL_INVALID, r.name)
	}

	return nil
}

type DateRule struct {
	name   string
	layout string
}

func Date(name string) *DateRule {
	return &DateRule{
		name:   name,
		layout: "2006-01-02",
	}
}

/**
* Layout
* @param layout string
* @return *DateRule
**/
func (r *DateRule) Layout(layout string) *DateRule {
	r.layout = layout
	return r
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *DateRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	str, ok := v.(string)
	if !ok {
		return fmt.Errorf(msg.MSG_STRING_REQUIRED, r.name)
	}

	_, err := time.Parse(r.layout, str)
	if err != nil {
		return fmt.Errorf(msg.MSG_DATE_INVALID, r.name, r.layout)
	}

	return nil
}

type EnumRule struct {
	name   string
	values map[string]struct{}
}

func Enum(name string, vals ...string) *EnumRule {
	m := map[string]struct{}{}
	for _, v := range vals {
		m[v] = struct{}{}
	}

	return &EnumRule{
		name:   name,
		values: m,
	}
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *EnumRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	str, ok := v.(string)
	if !ok {
		return fmt.Errorf(msg.MSG_STRING_REQUIRED, r.name)
	}

	if _, ok := r.values[str]; !ok {
		return fmt.Errorf(msg.MSG_ENUM_INVALID, r.name)
	}

	return nil
}

type ObjectRule struct {
	name  string
	rules []Rule
}

func Object(name string, rules ...Rule) *ObjectRule {
	return &ObjectRule{
		name:  name,
		rules: rules,
	}
}

func (r *ObjectRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	obj, ok := v.(map[string]interface{})
	if !ok {
		return fmt.Errorf(msg.MSG_OBJECT_REQUIRED, r.name)
	}

	js := Json(obj)

	for _, rule := range r.rules {
		if err := rule.Validate(js); err != nil {
			return err
		}
	}

	return nil
}

// PhoneRule validates that a field contains a valid mobile phone number.
// By default it enforces E.164 format: + followed by 7–15 digits (e.g. +573001234567).
// Use CountryCode to restrict to a specific country calling code prefix (e.g. "+57").
type PhoneRule struct {
	name        string
	countryCode string
}

func Phone(name string) *PhoneRule {
	return &PhoneRule{name: name}
}

/**
* CountryCode restricts the phone number to a given country calling code prefix.
* @param code string — e.g. "+57", "+1", "+34"
* @return *PhoneRule
**/
func (r *PhoneRule) CountryCode(code string) *PhoneRule {
	r.countryCode = code
	return r
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *PhoneRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	str, ok := v.(string)
	if !ok {
		return fmt.Errorf(msg.MSG_STRING_REQUIRED, r.name)
	}

	if !rePhone.MatchString(str) {
		return fmt.Errorf(msg.MSG_PHONE_INVALID, r.name)
	}

	if r.countryCode != "" {
		if len(str) < len(r.countryCode) || str[:len(r.countryCode)] != r.countryCode {
			return fmt.Errorf(msg.MSG_PHONE_INVALID, r.name)
		}
	}

	return nil
}

type BetweenRule struct {
	name string
	min  float64
	max  float64
}

/**
* Between
* @param name string, min, max float64
* @return *BetweenRule
**/
func Between(name string, min, max float64) *BetweenRule {
	return &BetweenRule{name, min, max}
}

/**
* Validate
* @param j Json
* @return error
**/
func (r *BetweenRule) Validate(j Json) error {
	v, ok := j[r.name]
	if !ok {
		return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, r.name)
	}

	var num float64

	switch t := v.(type) {
	case float64:
		num = t
	case int:
		num = float64(t)
	default:
		return fmt.Errorf(msg.MSG_NUMBER_REQUIRED, r.name)
	}

	if num < r.min || num > r.max {
		return fmt.Errorf(msg.MSG_NUMBER_BETWEEN, r.name, r.min, r.max)
	}

	return nil
}
