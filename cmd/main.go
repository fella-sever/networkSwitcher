package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"networkSwitcher/domain"
	"sync"
)

func main() {
	var set domain.MetricsCount
	wg := sync.WaitGroup{}
	wg.Add(2)
	r := gin.Default()
	go func() {
		r.GET("/", func(c *gin.Context) {
			jset, _ := json.Marshal(set)
			fmt.Println(string(jset))
			c.JSON(http.StatusOK, set)
		})
		r.POST("/set_threshold", func(c *gin.Context) {
			var newSettings domain.MetricsSetDto
			if err := c.BindJSON(&newSettings); err != nil {
				return
			}
			set.PacketLossSettings = newSettings.PacketLoss
			set.RttSettings = newSettings.RttSettings

			c.IndentedJSON(http.StatusCreated, set)

		})
		r.Run()
		wg.Done()
	}()

	go func() {
		for {
			err := set.PacketLossRttCount()

			if err != nil {
				log.Fatal(err)
			}
		}

		wg.Done()
	}()

	wg.Wait()
}
