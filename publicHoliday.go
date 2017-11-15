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

const apiUrl = "http://data.gov.ru/api/json/dataset/7708660670-proizvcalendar/version/20151123T183036/content?"

type PublicHoliday struct {
	token string

	lastUpdate time.Time
	cacheTime  time.Duration

	// data is map[year]map[month][]days
	dataWeekend  map[int]map[int][]int
	dataShortday map[int]map[int][]int

	workingDays         map[int]int64
	holidays            map[int]int64
	workingHours40hWeek map[int]float64
	workingHours36hWeek map[int]float64
	workingHours24hWeek map[int]float64

	mu sync.RWMutex
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

func New(accessToken string) *PublicHoliday {
	ph := new(PublicHoliday)
	ph.token = accessToken
	ph.cacheTime = time.Hour * 24
	return ph
}

func (ph *PublicHoliday) SetCacheTime(duration time.Duration) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.cacheTime = duration
}

func (ph *PublicHoliday) WorkingDays(date time.Time) (days int, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.workingDays[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return int(ph.workingDays[date.Year()]), nil
}

func (ph *PublicHoliday) Holidays(date time.Time) (days int, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.holidays[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return int(ph.holidays[date.Year()]), nil
}

func (ph *PublicHoliday) WorkingHours24hWeek(date time.Time) (hour float64, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.workingHours24hWeek[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return ph.workingHours24hWeek[date.Year()], nil
}

func (ph *PublicHoliday) WorkingHours36hWeek(date time.Time) (hour float64, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.workingHours36hWeek[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return ph.workingHours36hWeek[date.Year()], nil
}

func (ph *PublicHoliday) WorkingHours40hWeek(date time.Time) (hour float64, err error) {
	err = ph.chkUpdate()
	if err != nil {
		return
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.workingHours40hWeek[date.Year()]; !ok {
		return 0, errors.New("there is no data for this year")
	}
	return ph.workingHours40hWeek[date.Year()], nil
}

func (ph *PublicHoliday) IsWeekend(date time.Time) (bool, error) {
	err := ph.chkUpdate()
	if err != nil {
		return false, err
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.dataWeekend[date.Year()]; !ok {
		return false, errors.New("there is no data for this year")
	}
	if weekend, ok := ph.dataWeekend[date.Year()][int(date.Month())]; ok {
		for _, d := range weekend {
			if date.Day() == d {
				return true, nil
			}
		}
	}
	return false, nil
}

func (ph *PublicHoliday) IsShortDay(date time.Time) (bool, error) {
	err := ph.chkUpdate()
	if err != nil {
		return false, err
	}
	ph.mu.RLock()
	defer ph.mu.RUnlock()
	if _, ok := ph.dataShortday[date.Year()]; !ok {
		return false, errors.New("there is no data for this year")
	}
	if shortday, ok := ph.dataShortday[date.Year()][int(date.Month())]; ok {
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
	r, err := client.Get(apiUrl + params.Encode())
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

	ph.workingDays = map[int]int64{}
	ph.holidays = map[int]int64{}
	ph.workingHours24hWeek = map[int]float64{}
	ph.workingHours36hWeek = map[int]float64{}
	ph.workingHours40hWeek = map[int]float64{}
	ph.dataWeekend = map[int]map[int][]int{}
	ph.dataShortday = map[int]map[int][]int{}
	for i := range prcal {
		year, err := strconv.Atoi(prcal[i].Year)
		if err != nil {
			return err
		}

		ph.workingDays[year], err = strconv.ParseInt(prcal[i].WorkingDays, 10, 16)
		if err != nil {
			return err
		}

		ph.holidays[year], err = strconv.ParseInt(prcal[i].Holidays, 10, 16)
		if err != nil {
			return err
		}

		ph.workingHours24hWeek[year], err = strconv.ParseFloat(prcal[i].WorkingHours24hWeek, 16)
		if err != nil {
			return err
		}

		ph.workingHours36hWeek[year], err = strconv.ParseFloat(prcal[i].WorkingHours36hWeek, 16)
		if err != nil {
			return err
		}

		ph.workingHours40hWeek[year], err = strconv.ParseFloat(prcal[i].WorkingHours40hWeek, 16)
		if err != nil {
			return err
		}

		ph.dataWeekend[year] = map[int][]int{}
		ph.dataShortday[year] = map[int][]int{}

		ph.dataWeekend[year][1], ph.dataShortday[year][1], err = convDays(prcal[i].Jan)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][2], ph.dataShortday[year][2], err = convDays(prcal[i].Feb)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][3], ph.dataShortday[year][3], err = convDays(prcal[i].Mar)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][4], ph.dataShortday[year][4], err = convDays(prcal[i].Apr)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][5], ph.dataShortday[year][5], err = convDays(prcal[i].May)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][6], ph.dataShortday[year][6], err = convDays(prcal[i].Jun)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][7], ph.dataShortday[year][7], err = convDays(prcal[i].Jul)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][8], ph.dataShortday[year][8], err = convDays(prcal[i].Aug)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][9], ph.dataShortday[year][9], err = convDays(prcal[i].Sep)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][10], ph.dataShortday[year][10], err = convDays(prcal[i].Oct)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][11], ph.dataShortday[year][11], err = convDays(prcal[i].Nov)
		if err != nil {
			return err
		}
		ph.dataWeekend[year][12], ph.dataShortday[year][12], err = convDays(prcal[i].Dec)
		if err != nil {
			return err
		}

	}

	ph.lastUpdate = time.Now()
	return nil
}

func convDays(days string) (weekend []int, shortday []int, err error) {
	daysStr := strings.Split(days, ",")
	for d := range daysStr {
		var n int64
		if daysStr[d][len(daysStr[d])-1:] == "*" {
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
