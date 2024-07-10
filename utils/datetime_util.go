package utils

import (
	db "LeafMS-BackEnd/database"
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var daysInMonth = map[int]int{
	1:  31,
	2:  28,
	3:  31,
	4:  30,
	5:  31,
	6:  30,
	7:  31,
	8:  31,
	9:  30,
	10: 31,
	11: 30,
	12: 31,
}

func isLeapYear(year int) bool {
	if year%400 == 0 {
		return true
	} else if year%4 == 0 && year%100 != 0 {
		return true
	}
	return false
}

func FeasibleDate(date db.Datetime) error {
	day := date.Day
	month := date.Month
	year := date.Year

	if month > 12 {
		errStr := "The month provided is more than what is practically possible.\n"
		errStr += "i.e - The month is 13th or more than 13th, which is not possible."
		err := errors.New(errStr)
		return err
	} else if (isLeapYear(year) && month == 2 && day > (daysInMonth[month]+1)) || (!isLeapYear(year) && day > daysInMonth[month]) {
		errStr := "The number of days is more than possible for the month in the date.\n"
		err := errors.New(errStr)
		return err
	}
	return nil
}

func rollBackLeaveOneDay(date db.Datetime) db.Datetime {
	if date.Day == 1 {
		date.Day = daysInMonth[date.Month-1]
		if date.Month == 1 {
			date.Year -= 1
			date.Month = 12
		} else {
			date.Month -= 1
			if date.Month == 2 && isLeapYear(date.Year) {
				date.Day += 1
			}
		}
	} else {
		date.Day -= 1
	}
	return date
}

func rollForwardLeaveOneDay(date db.Datetime) db.Datetime {
	if date.Day == daysInMonth[date.Month] {
		date.Day = 1
		if date.Month == 12 {
			date.Year += 1
			date.Month = 1
		} else {
			date.Month += 1
		}
	} else {
		date.Day += 1
	}
	return date
}

func RemoveHolidayFromLeaveData(leave db.LeaveData) ([]db.LeaveData, error) {
	var splitLeaves []db.LeaveData
	leaveStartDate, err := ParseStringToDate(leave.Start)
	if err != nil {
		log.Println("There was problem parsing the starting date of a leave Err:", err)
		return []db.LeaveData{}, err
	}
	if err = FeasibleDate(leaveStartDate); err != nil {
		log.Println("The start date is not practically possible in the real world. Err: ", err)
		return []db.LeaveData{}, err
	}
	leaveEndDate, err := ParseStringToDate(leave.End)
	if err != nil {
		log.Println("There was problem parsing the ending date of a leave Err:", err)
		return []db.LeaveData{}, err
	}
	if err = FeasibleDate(leaveEndDate); err != nil {
		log.Println("The start date is not practically possible in the real world. Err: ", err)
		return []db.LeaveData{}, err
	}

	holidaysBson, err := database.Find("publicHolidays", bson.D{
		{Key: "$and", Value: bson.M{
			"date.year": bson.M{
				"$gte": leaveStartDate.Year,
				"$lte": leaveEndDate.Year,
			},
			"date.month": bson.M{
				"$gte": leaveStartDate.Month,
				"$lte": leaveEndDate.Month,
			},
			"date.day": bson.M{
				"$gte": leaveStartDate.Day,
				"$lte": leaveEndDate.Day,
			},
		}},
	})
	if err != nil {
		errMessage := "For fuck's sake, there was a problem, "
		errMessage += "while trying to find a holiday conflicting with the applied leave in the database. Err:"
		log.Println(errMessage, err)
		return []db.LeaveData{}, err
	}
	holidays := ReturnHolidays(holidaysBson)

	//	this part needs to be modified to include the case
	//	when there is no holiday conflict
	startDate := leave.Start
	for _, holiday := range holidays {
		var leaveSpan db.LeaveData
		leaveSpan.Id = primitive.NewObjectID()
		leaveSpan.Start = startDate
		leaveSpan.End = ParseDateToString(rollBackLeaveOneDay(holiday.Date.Datetime))
		splitLeaves = append(splitLeaves, leaveSpan)
		startDate = ParseDateToString(rollForwardLeaveOneDay(holiday.Date.Datetime))
	}
	lastLeaveSpan := db.LeaveData{
		Id:    primitive.NewObjectID(),
		Start: startDate,
		End:   leave.End,
	}
	splitLeaves = append(splitLeaves, lastLeaveSpan)
	return splitLeaves, nil
}
