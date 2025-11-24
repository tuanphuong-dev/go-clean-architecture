package validator

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type Validator interface {
	Engine() any
	ValidateStruct(obj any) error
	GetTranslator(locale string) (ut.Translator, error)
}

var (
	defaultValidator Validator
	vOnce            sync.Once
)

func DefaultValidator() Validator {
	vOnce.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

func RegisterValidatorWithGin() {
	binding.Validator = DefaultValidator().(*validatorImpl)
}

// Make sure validatorImpl implements both our Validator interface and Gin's StructValidator interface
var _ Validator = (*validatorImpl)(nil)
var _ binding.StructValidator = (*validatorImpl)(nil)

func New() Validator {
	v := new(validatorImpl)
	v.validate = validator.New()
	v.validate.SetTagName("binding")
	v.locale = "en" // default locale

	// Initialize universal translator
	v.initTranslator()

	v.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validations
	for _, validation := range defaultRegistrations {
		if err := v.validate.RegisterValidation(validation.Tag, validation.Func); err != nil {
			log.Fatalf("register validation %s error: %v", validation.Tag, err)
		}
	}

	// Register custom translations
	v.registerCustomTranslations()
	return v
}

type validatorImpl struct {
	validate   *validator.Validate
	uni        *ut.UniversalTranslator
	translator ut.Translator
	locale     string
}

// ValidateStruct implements both our Validator interface and Gin's StructValidator interface
func (v *validatorImpl) ValidateStruct(obj any) error {
	if kindOfData(obj) == reflect.Struct {
		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}
	return nil
}

func (v *validatorImpl) Engine() any {
	return v.validate
}

func (v *validatorImpl) GetTranslator(locale string) (ut.Translator, error) {
	trans, found := v.uni.GetTranslator(locale)
	if !found {
		return nil, fmt.Errorf("translator for locale '%s' not found", locale)
	}
	return trans, nil
}

func kindOfData(data any) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}
