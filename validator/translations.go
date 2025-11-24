package validator

import (
	"log"

	enLocale "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

func (v *validatorImpl) initTranslator() {
	// Create universal translator with supported locales
	en := enLocale.New()
	v.uni = ut.New(en, en)

	// Get default translator (English)
	trans, _ := v.uni.GetTranslator("en")
	v.translator = trans

	// Register default translations for English
	if err := en_translations.RegisterDefaultTranslations(v.validate, trans); err != nil {
		log.Printf("Failed to register English translations: %v", err)
	}
}

func (v *validatorImpl) registerCustomTranslations() {
	// Register English translations
	v.registerEnglishTranslations()

	// Add other languages here as needed
	// Example:
	// v.registerVietnamTranslations()
}

func (v *validatorImpl) registerEnglishTranslations() {
	trans, ok := v.uni.GetTranslator("en")
	if !ok {
		panic("Translator for 'en' not found")
	}

	translations := map[string]string{
		"action":              "{0} must be a valid action (register, login, reset_password, delete_account)",
		"otp_delivery_method": "{0} must be a valid OTP delivery method (zalo)",
		"service_type":        "{0} must be a valid order type (delivery, intercity_taxi)",
		"phone_number":        "{0} must be a valid phone number in E164 format",
		"identifier":          "{0} must be a valid email or phone number",
		"role":                "{0} must be a valid role",
		"date_of_birth":       "{0} must be a valid date of birth (in the past)",
		"id_card_number":      "{0} must be a valid ID card number (9 or 12 digits)",
		"license_expiry_date": "{0} must be a valid license expiry date (in the future)",
		"vehicle_type":        "{0} must be a valid vehicle type (car, motorbike)",
		"not_empty":           "{0} cannot be empty",
	}

	for tag, message := range translations {
		err := v.validate.RegisterTranslation(tag, trans,
			func(ut ut.Translator) error {
				return ut.Add(tag, message, true)
			},
			func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T(tag, fe.Field())
				return t
			},
		)
		if err != nil {
			log.Printf("Failed to register English translation for %s: %v", tag, err)
		}
	}
}
