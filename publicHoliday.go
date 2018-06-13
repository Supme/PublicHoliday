package publicHoliday

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const apiURL = "http://data.gov.ru/api/json/dataset/7708660670-proizvcalendar/version/20151123T183036/content?"

// PublicHoliday contains internal data
type PublicHoliday struct {
	token      string
	lastUpdate time.Time
	cacheTime  time.Duration
	data       map[int]publicHolidayData // data is map[year]
	mu         sync.RWMutex
}

type publicHolidayData struct {
	weekend             map[int][]int // map[month][]days
	shortday            map[int][]int // map[month][]days
	workdays            int64
	holidays            int64
	workingHours40hWeek float64
	workingHours36hWeek float64
	workingHours24hWeek float64
}

type proizvcalendar struct {
	Year                string `json:"Год/Месяц"`
	Jan                 string `json:"Январь"`
	Feb                 string `json:"Февраль"`
	Mar                 string `json:"Март"`
	Apr                 string `json:"Апрель"`
	May                 string `json:"Май"`
	Jun                 string `json:"Июнь"`
	Jul                 string `json:"Июль"`
	Aug                 string `json:"Август"`
	Sep                 string `json:"Сентябрь"`
	Oct                 string `json:"Октябрь"`
	Nov                 string `json:"Ноябрь"`
	Dec                 string `json:"Декабрь"`
	WorkingDays         string `json:"Всего рабочих дней"`
	Holidays            string `json:"Всего праздничных и выходных дней"`
	WorkingHours40hWeek string `json:"Количество рабочих часов при 40-часовой рабочей неделе"`
	WorkingHours36hWeek string `json:"Количество рабочих часов при 36-часовой рабочей неделе"`
	WorkingHours24hWeek string `json:"Количество рабочих часов при 24-часовой рабочей неделе"`
}

// New takes access token and return a prepared instance of PublicHoliday
func New(accessToken string) *PublicHoliday {
	ph := new(PublicHoliday)
	ph.token = accessToken
	ph.cacheTime = time.Hour * 24
	return ph
}

// SetCacheTime set the lifetime of the obtained data from the source
func (ph *PublicHoliday) SetCacheTime(duration time.Duration) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.cacheTime = duration
}

// WorkingDays returns the number of working days in the year received at the entrance
func (ph *PublicHoliday) WorkingDays(date time.Time) (days int, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return int(ph.data[date.Year()].workdays), nil
}

// Holidays returns the number of holidays in the year received at the entrance
func (ph *PublicHoliday) Holidays(date time.Time) (days int, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return int(ph.data[date.Year()].holidays), nil
}

// WorkingHours24hWeek returns the number of working hours at 24 hour week in the year received at the entrance
func (ph *PublicHoliday) WorkingHours24hWeek(date time.Time) (hour float64, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return ph.data[date.Year()].workingHours24hWeek, nil
}

// WorkingHours36hWeek returns the number of working hours at 36 hour week in the year received at the entrance
func (ph *PublicHoliday) WorkingHours36hWeek(date time.Time) (hour float64, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return ph.data[date.Year()].workingHours36hWeek, nil
}

// WorkingHours40hWeek returns the number of working hours at 40 hour week in the year received at the entrance
func (ph *PublicHoliday) WorkingHours40hWeek(date time.Time) (hour float64, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return ph.data[date.Year()].workingHours40hWeek, nil
}

// IsWeekend returns whether it is true that the weekend
func (ph *PublicHoliday) IsWeekend(date time.Time) (bool, error) {
	err := ph.chkUpdate()
	if err != nil {
		return false, err
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return false, errors.New("there is no data for this year")
	}
	if weekend, ok := ph.data[date.Year()].weekend[int(date.Month())]; ok {
		for _, d := range weekend {
			if date.Day() == d {
				return true, nil
			}
		}
	}
	return false, nil
}

// IsShortDay returns whether it is true that the short day
func (ph *PublicHoliday) IsShortDay(date time.Time) (bool, error) {
	err := ph.chkUpdate()
	if err != nil {
		return false, err
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.data[date.Year()]; !ok {
		return false, errors.New("there is no data for this year")
	}
	if shortday, ok := ph.data[date.Year()].shortday[int(date.Month())]; ok {
		for _, d := range shortday {
			if date.Day() == d {
				return true, nil
			}
		}
	}
	return false, nil
}

func (ph *PublicHoliday) chkUpdate() error {
	if time.Now().Unix() > ph.lastUpdate.Add(ph.cacheTime).Unix() {
		err := ph.Update()
		if err != nil {
			return err
		}
	}
	return nil
}

// Update manual update data from the source
func (ph *PublicHoliday) Update() error {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	var client = &http.Client{
		Timeout: time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	params := url.Values{}
	params.Add("access_token", ph.token)
	r, err := client.Get(apiURL + params.Encode())
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK && (ph.lastUpdate != time.Time{}) {
		return nil
	}

	var prcal []proizvcalendar
	err = json.NewDecoder(r.Body).Decode(&prcal)
	if err != nil {
		return err
	}

	tmphs := make(map[int]publicHolidayData)
	var errs []error
	for i := range prcal {
		year, err := strconv.Atoi(prcal[i].Year)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		var tmph publicHolidayData

		tmph.workdays, err = strconv.ParseInt(prcal[i].WorkingDays, 10, 16)
		errs = append(errs, err)
		tmph.holidays, err = strconv.ParseInt(prcal[i].Holidays, 10, 16)
		errs = append(errs, err)
		tmph.workingHours24hWeek, err = strconv.ParseFloat(prcal[i].WorkingHours24hWeek, 16)
		errs = append(errs, err)
		tmph.workingHours36hWeek, err = strconv.ParseFloat(prcal[i].WorkingHours36hWeek, 16)
		errs = append(errs, err)
		tmph.workingHours40hWeek, err = strconv.ParseFloat(prcal[i].WorkingHours40hWeek, 16)
		errs = append(errs, err)

		tmph.weekend = map[int][]int{}
		tmph.shortday = map[int][]int{}
		tmph.weekend[1], tmph.shortday[1], err = convDays(prcal[i].Jan)
		errs = append(errs, err)
		tmph.weekend[2], tmph.shortday[2], err = convDays(prcal[i].Feb)
		errs = append(errs, err)
		tmph.weekend[3], tmph.shortday[3], err = convDays(prcal[i].Mar)
		errs = append(errs, err)
		tmph.weekend[4], tmph.shortday[4], err = convDays(prcal[i].Apr)
		errs = append(errs, err)
		tmph.weekend[5], tmph.shortday[5], err = convDays(prcal[i].May)
		errs = append(errs, err)
		tmph.weekend[6], tmph.shortday[6], err = convDays(prcal[i].Jun)
		errs = append(errs, err)
		tmph.weekend[7], tmph.shortday[7], err = convDays(prcal[i].Jul)
		errs = append(errs, err)
		tmph.weekend[8], tmph.shortday[8], err = convDays(prcal[i].Aug)
		errs = append(errs, err)
		tmph.weekend[9], tmph.shortday[9], err = convDays(prcal[i].Sep)
		errs = append(errs, err)
		tmph.weekend[10], tmph.shortday[10], err = convDays(prcal[i].Oct)
		errs = append(errs, err)
		tmph.weekend[11], tmph.shortday[11], err = convDays(prcal[i].Nov)
		errs = append(errs, err)
		tmph.weekend[12], tmph.shortday[12], err = convDays(prcal[i].Dec)
		errs = append(errs, err)

		tmphs[year] = tmph
	}

	for e := range errs {
		if errs[e] != nil {
			return errs[e] // ToDo join all errors
		}
	}

	ph.data = tmphs
	ph.lastUpdate = time.Now()
	return nil
}

func convDays(days string) (weekend []int, shortday []int, err error) {
	daysStr := strings.Split(days, ",")
	for d := range daysStr {
		var n int64
		if daysStr[d][len(daysStr[d])-1:] == "*" || daysStr[d][len(daysStr[d])-1:] == "+" {
			n, err = strconv.ParseInt(daysStr[d][:len(daysStr[d])-1], 10, 8)
			if err != nil {
				return
			}
			shortday = append(shortday, int(n))
		} else {
			n, err = strconv.ParseInt(daysStr[d], 10, 8)
			if err != nil {
				return
			}
			weekend = append(weekend, int(n))
		}
	}
	return
}
