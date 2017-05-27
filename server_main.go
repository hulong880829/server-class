package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tabalt/gracehttp"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

//mysql 数据库连接池
var Pool_Mysql *sql.DB

type Student struct {
	Id                 string
	ClassNumber, Score int
}

func main() {

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	std_path := strings.Replace(dir, "\\", "/", -1) + "/StdLog/std.log"
	logFile, _ := os.OpenFile(std_path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0755)
	if os.Getppid() != 1 { //判断当其是否是子进程，当父进程return之后，子进程会被 系统1 号进程接管
		filePath, _ := filepath.Abs(os.Args[0]) //将命令行参数中执行文件路径转换成可用路径
		cmd := exec.Command(filePath, os.Args[1:]...)
		//将其他命令传入生成出的进程
		cmd.Stdin = os.Stdin //给新进程设置文件描述符，可以重定向到文件中
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Start() //开始执行新进程，不等待新进程退出
		return
	}
	Pool_Mysql, _ = sql.Open("mysql", "root:hul@tcp(127.0.0.1:3306)/test?charset=utf8")
	Pool_Mysql.SetMaxOpenConns(30)
	Pool_Mysql.SetMaxIdleConns(10)
	Pool_Mysql.Ping()

	_, err := Pool_Mysql.Exec("create table if not exists `Student_Info`(`id`  varchar(6) NOT NULL DEFAULT '' ,`classNumber`  tinyint(2) NOT NULL ,`score`  tinyint(3) NOT NULL ,PRIMARY KEY (`id`));")
	_, err = Pool_Mysql.Exec("create table if not exists `Class_Info`(`classNumber`  tinyint(2) NOT NULL ,`teacher_name`  varchar(20) NOT NULL DEFAULT '',PRIMARY KEY (`classNumber`));")
	if err != nil {
		fmt.Println("Init Server Err!", err)
		return
	}
	http.HandleFunc("/register-student", Handle_Register_Student)     //注册学生
	http.HandleFunc("/register-class", Handle_Register_Class)         //注册班级
	http.HandleFunc("/get-class-total-score", Handle_Get_Class_Score) //查询学生所在班级总分
	http.HandleFunc("/get-top-teacher", Handle_Get_Top_Teacher_Name)  //查询最高分学生班级老师姓名

	port_str := fmt.Sprintf(":%d", 80)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := gracehttp.ListenAndServe(port_str, nil) //设置监听的端口
		if err != nil {
			os.Exit(-1)
		}
	}()
	wg.Wait()

}

func Get_Http_ParseFormArg(r *http.Request, key string) string {
	if len(r.Form[key]) > 0 {
		return r.Form[key][0]
	}
	return ""
}
func Get_Interface_Int(e interface{}) (int, error) {
	var res int
	var err error = nil
	switch v := e.(type) {
	case int:
		res = v
		break
	case float64:
		res = int(v)
		break
	default:
		err = errors.New("interface type err")
	}
	return res, err
}
func Get_Interface_String(e interface{}) (string, error) {
	var res string
	var err error = nil
	switch v := e.(type) {
	case string:
		res = v
		break
	default:
		err = errors.New("interface type err")
	}
	return res, err
}

func Http_Response(w http.ResponseWriter, res string) {
	w.Write([]byte(res))
}

//检测数据是否正确 1 学生ID  2，班级ID  3，学生分数
func Check_Data(data string, n_type int) bool {
	res := true
	switch n_type {
	case 1:
		if len(data) != 5 {
			res = false
			break
		}
		_, err := strconv.Atoi(data)
		if err != nil {
			res = false
			break
		}
	case 2:
		tmp_int, err := strconv.Atoi(data)
		if err != nil || tmp_int > 99 || tmp_int < 0 {
			res = false
			break
		}
	case 3:
		tmp_int, err := strconv.Atoi(data)
		if err != nil || tmp_int > 100 || tmp_int < 0 {
			res = false
			break
		}
	case 4:
		if data == "" || len(data) > 20 {
			res = false
			break
		}

	}
	return res
}

func Handle_Register_Student(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Http_Response(w, "[Register Failed] http err!")
		return
	}
	json_map := make(map[string]interface{})
	json.Unmarshal([]byte(body), &json_map)

	if r.Method != "POST" {
		Http_Response(w, "[Register Failed] api method err!")
		return
	}
	//获取id
	value_id, ok := json_map["id"]
	id, id_err := Get_Interface_String(value_id)
	if Check_Data(id, 1) == false || id_err != nil || ok == false {
		Http_Response(w, "[Register Failed] student id  err!")
		return
	}
	//获取班级信息
	value_number, ok := json_map["classNumber"]
	class_number, num_err := Get_Interface_Int(value_number)
	if Check_Data(strconv.Itoa(class_number), 2) == false || ok == false || num_err != nil {
		Http_Response(w, "[Register Failed] student classNumber err!")
		return
	}
	//获取学分
	value_score, ok := json_map["score"]
	score, s_err := Get_Interface_Int(value_score)
	if Check_Data(strconv.Itoa(score), 3) == false || s_err != nil || ok == false {
		Http_Response(w, "[Register Failed] student core  err!")
		return
	}
	sql_str := fmt.Sprintf("replace into `Student_Info` (`id`,`classNumber`,`score`) values('%s',%d,%d)", id, class_number, score)
	_, err = Pool_Mysql.Exec(sql_str)
	if err != nil {
		err_str := fmt.Sprintf("[Register Failed] sql err[%s]", sql_str)
		Http_Response(w, err_str)
		return
	}
	res_str := fmt.Sprintf("register ok![id:%s,classNumber:%d,score:%d]", id, class_number, score)
	Http_Response(w, res_str)
	return
}

func Handle_Register_Class(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Http_Response(w, "[Register Failed] http err!")
		return
	}
	json_map := make(map[string]interface{})
	json.Unmarshal([]byte(body), &json_map)

	if r.Method != "POST" {
		Http_Response(w, "[Register Failed] api method err!")
		return
	}
	//获取班级信息
	value_number, ok := json_map["classNumber"]
	class_number, num_err := Get_Interface_Int(value_number)
	if Check_Data(strconv.Itoa(class_number), 2) == false || ok == false || num_err != nil {
		Http_Response(w, "[Register Failed] classNumber err!")
		return
	}
	//获取教师信息
	value_teacher, ok := json_map["teacher"]
	teacher, t_err := Get_Interface_String(value_teacher)
	if Check_Data(teacher, 4) == false || t_err != nil || ok == false {
		Http_Response(w, "[Register Failed] teacher  err!")
		return
	}
	sql_str := fmt.Sprintf("replace into `Class_Info` (`classNumber`,`teacher_name`) values(%d,'%s')", class_number, teacher)
	_, err = Pool_Mysql.Exec(sql_str)
	if err != nil {
		err_str := fmt.Sprintf("[Register Failed] sql err[%s]", sql_str)
		Http_Response(w, err_str)
		return
	}
	res_str := fmt.Sprintf("register ok![classNumber:%d,teacher:%s]", class_number, teacher)
	Http_Response(w, res_str)
	return
}

func Handle_Get_Class_Score(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.Method != "GET" {
		Http_Response(w, "[Query Failed] api method err!")
		return
	}
	id := Get_Http_ParseFormArg(r, "student_id")

	sql_str := fmt.Sprintf("select sum(score) from Student_Info where classNumber = (select classNumber from Student_Info where id='%s')", id)
	rows, err := Pool_Mysql.Query(sql_str)
	if err != nil {
		err_str := fmt.Sprintf("[Register Failed] sql err[%s]", sql_str)
		Http_Response(w, err_str)
		return
	}
	defer rows.Close()
	var total_score int = -1
	for rows.Next() {
		rows.Scan(&total_score)

	}
	if total_score != -1 {
		send_json := make(map[string]interface{})
		send_json["total"] = total_score
		send_byte, _ := json.Marshal(send_json)
		Http_Response(w, string(send_byte))

	} else {
		send_json := make(map[string]interface{})
		send_json["error"] = "student-not-found"
		send_byte, _ := json.Marshal(send_json)
		Http_Response(w, string(send_byte))
	}

	return

}

func Handle_Get_Top_Teacher_Name(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.Method != "GET" {
		Http_Response(w, "[Query Failed] api method err!")
		return
	}

	sql_str := fmt.Sprintf("select teacher_name from Class_Info where classNumber =(select classNumber from Student_Info order by score desc,id asc limit 1)")
	rows, err := Pool_Mysql.Query(sql_str)
	if err != nil {
		err_str := fmt.Sprintf("[Query Failed] sql err[%s]", sql_str)
		Http_Response(w, err_str)
		return
	}
	defer rows.Close()
	var teacher string = ""
	for rows.Next() {
		rows.Scan(&teacher)
	}
	send_json := make(map[string]interface{})
	send_json["teacher"] = teacher
	send_byte, _ := json.Marshal(send_json)
	Http_Response(w, string(send_byte))

	return

}
