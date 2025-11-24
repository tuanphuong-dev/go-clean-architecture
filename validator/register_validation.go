package validator

import (
	"regexp"
	"time"

	"go-clean-arch/domain"
	"go-clean-arch/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

const IDCardNumberRegexString = `^\d+$`

type Registration struct {
	Tag  string
	Func validator.Func
}

var defaultRegistrations = [...]Registration{
	{
		Tag:  Action,
		Func: IsValidAction,
	},
	{
		Tag:  OTPDeliveryMethod,
		Func: IsValidOTPDeliveryMethod,
	},
	{
		Tag:  ServiceType,
		Func: IsValidServiceType,
	},
	{
		Tag:  PhoneNumber,
		Func: IsValidPhoneNumber,
	},
	{
		Tag:  Identifier,
		Func: IsValidIdentifier,
	},
	{
		Tag:  Role,
		Func: IsValidRole,
	},
	{
		Tag:  DateOfBirth,
		Func: IsValidDateOfBirth,
	},
	{
		Tag:  IDCardNumber,
		Func: IsValidIDNumber,
	},
	{
		Tag:  LicenseExpiryDate,
		Func: IsValidLicenseExpiryDate,
	},
	{
		Tag:  VehicleType,
		Func: IsValidVehicleType,
	},
	{
		Tag:  NotEmpty,
		Func: IsNotEmpty,
	},
	{
		Tag:  Email,
		Func: IsValidEmail,
	},
}

func IsValidAction(fl validator.FieldLevel) bool {
	input := domain.Action(fl.Field().String())
	switch input {
	case domain.Register,
		domain.Login,
		domain.ResetPassword,
		domain.DeleteAccount:
		return true
	default:
		return false
	}
}

func IsValidOTPDeliveryMethod(fl validator.FieldLevel) bool {
	input := domain.OTPDeliveryMethod(fl.Field().String())
	switch input {
	case domain.Zalo:
		return true
	default:
		return false
	}
}

func IsValidServiceType(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	return domain.ServiceType(input).IsValid()
}

/*TODO
- Refactor all phone numbers in system to be E164 format
*/

func IsValidPhoneNumber(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	if input == "" {
		// If it's optional and empty, consider it valid
		return true
	}

	e164, err := utils.FormatE164(input, utils.RegionVN)
	if err != nil {
		return false
	}
	return utils.IsE164Format(e164)
}

func IsValidEmail(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	return utils.IsEmail(input)
}

func IsValidIdentifier(fl validator.FieldLevel) bool {
	return IsValidEmail(fl) || IsValidPhoneNumber(fl)
}

func IsValidRole(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	return lo.Contains([]string{
		string(domain.RoleIDCustomer),
		string(domain.RoleIDGuest),
		string(domain.RoleIDAdmin),
		string(domain.RoleIDSystemAdmin),
	}, input)
}

func IsValidDateOfBirth(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(domain.Date)
	if !ok {
		return false
	}

	birthDate := time.Time(date)
	now := time.Now()

	// Check if date is in the past
	if birthDate.After(now) || birthDate.Equal(now) {
		return false
	}
	return true
}

func IsValidIDNumber(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	regex, err := regexp.Compile(IDCardNumberRegexString)
	if err != nil {
		return false
	}

	if !regex.MatchString(input) {
		return false
	}

	if len(input) != 9 && len(input) != 12 {
		return false
	}
	return true
}

func IsValidLicenseExpiryDate(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(domain.Date)
	if !ok {
		return false
	}

	expiryDate := time.Time(date)
	now := time.Now()

	// Check if date is in the past
	if expiryDate.Before(now) || expiryDate.Equal(now) {
		return false
	}
	return true
}

func IsValidVehicleType(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	return domain.VehicleType(input).IsValid()
}

func IsNotEmpty(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	return len(input) > 0
}
