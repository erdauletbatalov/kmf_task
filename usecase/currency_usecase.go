package usecase

import (
	"currency/domain"
	"currency/repository"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type CurrencyService struct {
	Repo           repository.CurrencyRepository
	ExternalClient repository.ExternalService
}

func NewCurrencyService(repo repository.CurrencyRepository, externalClient repository.ExternalService) *CurrencyService {
	return &CurrencyService{
		Repo:           repo,
		ExternalClient: externalClient,
	}
}

func (service *CurrencyService) SaveCurrency(date string) error {
	currencyData, err := service.ExternalClient.GetCurrencyData(date)
	if err != nil {
		return err
	}

	go func() {
		err := service.Repo.Save(currencyData)
		if err != nil {
			log.Println("Error saving currency data to database:", err)
		} else {
			log.Println("Currency data saved to database asynchronously")
		}
	}()

	return nil
}

func (service *CurrencyService) GetCurrency(date, code string) ([]domain.Currency, error) {
	return service.Repo.GetCurrency(date, code)
}

type nationalBankClient struct{}

func NewNationalBankClient() *nationalBankClient {
	return &nationalBankClient{}
}

func (nbc *nationalBankClient) GetCurrencyData(date string) ([]domain.Currency, error) {
	// Construct the URL with the provided date
	url := fmt.Sprintf("https://nationalbank.kz/rss/get_rates.cfm?fdate=%s", date)

	// Make a GET request to the National Bank API
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Decode XML response
	var rates struct {
		Items []struct {
			Title string `xml:"fullname"`
			Code  string `xml:"title"`
			Value string `xml:"description"`
			Date  string `xml:"date"`
		} `xml:"rates>item"`
	}
	if err := xml.Unmarshal(body, &rates); err != nil {
		return nil, err
	}

	// Convert XML data to Currency struct
	var currencies []domain.Currency
	for _, item := range rates.Items {
		value, err := strconv.ParseFloat(item.Value, 64)
		if err != nil {
			return nil, err
		}
		adate, err := time.Parse("02.01.2006", item.Date)
		if err != nil {
			return nil, err
		}
		currencies = append(currencies, domain.Currency{
			Title: item.Title,
			Code:  item.Code,
			Value: value,
			ADate: adate,
		})
	}

	return currencies, nil
}