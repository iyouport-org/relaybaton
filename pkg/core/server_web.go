package core

import (
	"bufio"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/iyouport-org/relaybaton/pkg/model"
	"github.com/iyouport-org/relaybaton/pkg/webapi"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (server *Server) serveWeb(ctx *fasthttp.RequestCtx) {
	memconn := server.ms.Dial()
	_, err := ctx.Request.WriteTo(memconn)
	if err != nil {
		log.Error(err)
		return
	}
	err = ctx.Response.Read(bufio.NewReader(memconn))
	if err != nil {
		log.Error(err)
		return
	}
}

func (server *Server) ServeRoot(c *gin.Context) {
	c.HTML(fasthttp.StatusOK, "index.html", gin.H{})
}

func (server *Server) GetCaptcha(c *gin.Context) {
	session := sessions.Default(c)
	_captchaID := session.Get("captchaID")
	captchaID, ok := _captchaID.(string)
	if !(ok && captcha.Reload(captchaID)) {
		captchaID = captcha.New()
		session.Set("captchaID", captchaID)
		err := session.Save()
		if err != nil {
			log.Error(err)
			return
		}
	}
	err := captcha.WriteImage(c.Writer, captchaID, captcha.StdWidth, captcha.StdHeight)
	if err != nil {
		log.Error(err)
	}
}

//register
func (server *Server) PostUser(c *gin.Context) {
	session := sessions.Default(c)
	request := &webapi.PostUserRequest{}
	err := c.BindJSON(request)
	if err != nil {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusInternalServerError, err)
		return
	}
	if request.Username == "admin" {
		c.JSON(fasthttp.StatusForbidden, &webapi.PostUserResponse{
			OK:       false,
			ErrorMsg: "Username existed",
		})
		return
	}
	captchaID := session.Get("captchaID")
	if captchaID != nil {
		if captcha.VerifyString(captchaID.(string), request.Captcha) {
			sha512key, err := base64.StdEncoding.DecodeString(request.Password)
			if err != nil {
				log.Error(err)
				c.AbortWithError(fasthttp.StatusInternalServerError, err)
				return
			}
			if len(sha512key) != 64 {
				c.JSON(fasthttp.StatusForbidden, &webapi.PostUserResponse{
					OK:       false,
					ErrorMsg: "Wrong password",
				})
				return
			}
			cryptKey, err := bcrypt.GenerateFromPassword(sha512key, bcrypt.DefaultCost)
			if err != nil {
				log.Error(err)
				c.AbortWithError(fasthttp.StatusInternalServerError, err)
				return
			}
			err = server.DB.DB.Create(&model.User{
				Username:    request.Username,
				Password:    base64.StdEncoding.EncodeToString(cryptKey),
				TrafficUsed: 0,
				PlanID:      1,
				PlanStart:   time.Now(),
				PlanReset:   time.Now(),
				PlanEnd:     time.Now(),
			}).Error
			if err != nil {
				log.Error(err)
				if err.Error() == "UNIQUE constraint failed: users.username" {
					c.JSON(fasthttp.StatusForbidden, &webapi.PostUserResponse{
						OK:       false,
						ErrorMsg: "Username existed",
					})
				}
				c.AbortWithError(fasthttp.StatusInternalServerError, err)
				return
			} else {
				c.JSON(fasthttp.StatusAccepted, &webapi.PostUserResponse{
					OK:       true,
					ErrorMsg: "",
				})
				return
			}
		} else {
			c.JSON(fasthttp.StatusForbidden, &webapi.PostUserResponse{
				OK:       false,
				ErrorMsg: "Wrong Captcha",
			})
			return
		}
	} else {
		c.AbortWithStatusJSON(fasthttp.StatusBadRequest, &webapi.PostUserResponse{
			OK:       false,
			ErrorMsg: "Wrong Session",
		})
		return
	}
}

func (server *Server) DeleteUser(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			result := server.DB.DB.Delete(&model.User{}, id)
			if result.Error == nil {
				c.JSON(fasthttp.StatusOK, id)
			} else {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) PutUser(c *gin.Context) {
	request := &webapi.User{}
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err == nil {
		err = c.BindJSON(request)

		if err == nil {
			log.Debug(request)
			result := server.DB.DB.Model(&model.User{
				Model: gorm.Model{
					ID: uint(id),
				},
			}).Updates(map[string]interface{}{
				"username":     request.Username,
				"role":         request.Role,
				"plan_id":      request.PlanID,
				"traffic_used": request.TrafficUsed,
				"plan_start":   request.PlanStart,
				"plan_reset":   request.PlanReset,
				"plan_end":     request.PlanEnd,
			})
			if result.Error != nil {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			} else {
				log.Debug(result.RowsAffected)
				c.JSON(fasthttp.StatusOK, request)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusBadRequest, err)
	}
}

func (server *Server) GetUser(c *gin.Context) {
	session := sessions.Default(c)
	_, ok := c.GetQuery("_start")
	if !ok {
		userID := session.Get("userID")
		if userID != nil {
			if userID == "admin" {
				c.JSON(fasthttp.StatusOK, &webapi.User{
					ID:                 0,
					Username:           "admin",
					Role:               model.RoleAdmin,
					PlanID:             0,
					PlanName:           "",
					PlanBandwidthLimit: 0,
					PlanTrafficLimit:   0,
					TrafficUsed:        0,
					PlanStart:          time.Now(),
					PlanReset:          time.Now(),
					PlanEnd:            time.Now(),
				})
			} else {
				userIDUint, ok := userID.(uint)
				if ok {
					user := &model.User{}
					result := server.DB.DB.First(user, userIDUint)
					if result.Error == nil {
						c.JSON(fasthttp.StatusOK, &webapi.User{
							ID:                 user.ID,
							Username:           user.Username,
							Role:               user.Role,
							PlanID:             user.PlanID,
							PlanName:           user.Plan.Name,
							PlanBandwidthLimit: user.Plan.BandwidthLimit,
							PlanTrafficLimit:   user.Plan.TrafficLimit,
							TrafficUsed:        user.TrafficUsed,
							PlanStart:          user.PlanStart,
							PlanReset:          user.PlanReset,
							PlanEnd:            user.PlanEnd,
						})
						return
					} else {
						log.Error(result.Error)
						c.AbortWithStatusJSON(fasthttp.StatusForbidden, &webapi.User{})
						return
					}
				} else {
					c.AbortWithStatusJSON(fasthttp.StatusForbidden, &webapi.User{})
					return
				}
			}
		} else {
			c.AbortWithStatusJSON(fasthttp.StatusForbidden, &webapi.User{})
			return
		}
	} else {
		server.GetUserList(c)
	}
}

func (server *Server) GetUserList(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		start := c.Query("_start")
		end := c.Query("_end")
		sort := c.Query("_sort")
		order := c.Param("_order")
		var modelUsers []model.User
		var total int64
		var contentRange int64
		result := server.DB.DB.Where("id BETWEEN ? and ?", start, end).Order(fmt.Sprintf("%s %s", sort, order)).Find(&modelUsers).Count(&total)
		if result.Error == nil {
			server.DB.DB.Table("users").Count(&contentRange)
			c.Writer.Header().Set("X-Total-Count", strconv.FormatInt(contentRange, 10))
			c.JSON(fasthttp.StatusOK, webapi.GetUsers(modelUsers))
		} else {
			c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) GetUserOne(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			modelUser := &model.User{}
			result := server.DB.DB.First(modelUser, id)
			if result.Error == nil {
				c.JSON(fasthttp.StatusOK,
					&webapi.User{
						ID:                 modelUser.ID,
						Username:           modelUser.Username,
						Role:               modelUser.Role,
						PlanID:             modelUser.PlanID,
						PlanName:           modelUser.Plan.Name,
						PlanBandwidthLimit: modelUser.Plan.BandwidthLimit,
						PlanTrafficLimit:   modelUser.Plan.TrafficLimit,
						TrafficUsed:        modelUser.TrafficUsed,
						PlanStart:          modelUser.PlanStart,
						PlanReset:          modelUser.PlanReset,
						PlanEnd:            modelUser.PlanEnd,
					},
				)
			} else {
				log.Error(err)
				c.AbortWithError(fasthttp.StatusBadRequest, err)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

//login
func (server *Server) PostSession(c *gin.Context) {
	session := sessions.Default(c)
	request := &webapi.PostSessionRequest{}
	err := c.BindJSON(request)
	if err != nil {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusBadRequest, err)
		return
	}
	sha512key, err := base64.StdEncoding.DecodeString(request.Password)
	if err != nil {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusBadRequest, err)
		return
	}
	if len(sha512key) != 64 {
		log.Debug(sha512key)
		c.JSON(fasthttp.StatusBadRequest, &webapi.PostSessionResponse{
			OK:       false,
			ErrorMsg: "Wrong password",
		})
		return
	}
	if request.Username == "admin" {
		correctKey := sha512.Sum512([]byte(server.ConfigGo.Server.AdminPassword))
		if string(correctKey[:]) == string(sha512key) {
			session.Set("userID", "admin")
		} else {
			log.WithFields(log.Fields{
				"password_in":          request.Password,
				"password_in_sha512":   sha512key,
				"real_password":        server.ConfigGo.Server.AdminPassword,
				"real_password_sha512": correctKey[:],
			}).Debug("admin login error")
		}
	} else {
		user := model.User{}
		result := server.DB.DB.Where("username = ?", request.Username).First(&user)
		err = result.Error
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
			return
		}
		correctKey, err := base64.StdEncoding.DecodeString(user.Password)
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
			return
		}
		err = bcrypt.CompareHashAndPassword(correctKey, sha512key)
		if err != nil {
			log.Error(err)
			c.JSON(fasthttp.StatusForbidden, &webapi.PostSessionResponse{
				OK:       false,
				ErrorMsg: "Wrong password",
			})
			return
		}
		session.Set("userID", user.ID)
	}
	err = session.Save()
	if err != nil {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusInternalServerError, err)
		return
	}
	c.JSON(fasthttp.StatusOK, &webapi.PostSessionResponse{
		OK:       true,
		ErrorMsg: "",
	})
}

func (server *Server) DeleteSession(c *gin.Context) {

}

func (server *Server) PutSession(c *gin.Context) {

}

func (server *Server) GetSession(c *gin.Context) {

}

func (server *Server) PostLog(c *gin.Context) {

}

func (server *Server) DeleteLog(c *gin.Context) {

}

func (server *Server) PutLog(c *gin.Context) {

}

func (server *Server) GetLogOne(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			modelLog := &model.Log{}
			result := server.DB.DB.First(modelLog, id)
			if result.Error == nil {
				c.JSON(fasthttp.StatusOK,
					&webapi.GetLogResponse{
						ID:       modelLog.ID,
						CreateAt: modelLog.CreatedAt,
						Level:    modelLog.Level,
						Func:     modelLog.Func,
						File:     modelLog.File,
						Msg:      modelLog.Msg,
						Stack:    modelLog.Stack,
						Fields:   modelLog.Fields,
					},
				)
			} else {
				log.Error(err)
				c.AbortWithError(fasthttp.StatusBadRequest, err)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) GetLogList(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		start := c.Query("_start")
		end := c.Query("_end")
		sort := c.Query("_sort")
		order := c.Param("_order")
		var modelLogs []model.Log
		var total int64
		var contentRange int64
		result := server.DB.DB.Where("id BETWEEN ? and ?", start, end).Order(fmt.Sprintf("%s %s", sort, order)).Find(&modelLogs).Count(&total)
		if result.Error == nil {
			server.DB.DB.Table("log").Count(&contentRange)
			c.Writer.Header().Set("X-Total-Count", strconv.FormatInt(contentRange, 10))
			c.JSON(fasthttp.StatusOK, webapi.GetLogs(modelLogs))
		} else {
			c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) PostConfig(c *gin.Context) {

}

func (server *Server) DeleteConfig(c *gin.Context) {

}

func (server *Server) PutConfig(c *gin.Context) {

}

func (server *Server) GetConfig(c *gin.Context) {

}

func (server *Server) PostPlan(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		request := &webapi.PostPlanRequest{}
		err := c.BindJSON(request)
		if err == nil {
			plan := model.Plan{
				Name:           request.Name,
				BandwidthLimit: request.BandwidthLimit,
				TrafficLimit:   request.TrafficLimit,
			}
			result := server.DB.DB.Save(&plan)
			if result.Error != nil {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			} else {
				c.JSON(fasthttp.StatusOK, webapi.GetPlan(plan))
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) DeletePlan(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			result := server.DB.DB.Delete(&model.Plan{}, id)
			if result.Error == nil {
				c.JSON(fasthttp.StatusOK, id)
			} else {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) PutPlan(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		request := &webapi.Plan{}
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			err = c.BindJSON(request)
			if err == nil {
				result := server.DB.DB.Model(&model.Plan{
					Model: gorm.Model{
						ID: uint(id),
					},
				}).Updates(model.Plan{
					Model: gorm.Model{
						UpdatedAt: time.Now(),
					},
					Name:           request.Name,
					BandwidthLimit: request.BandwidthLimit,
					TrafficLimit:   request.TrafficLimit,
				})
				if result.Error != nil {
					log.Error(result.Error)
					c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
				} else {
					c.JSON(fasthttp.StatusOK, request)
				}
			} else {
				log.Error(err)
				c.AbortWithError(fasthttp.StatusBadRequest, err)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) GetPlanOne(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err == nil {
		modelPlan := &model.Plan{}
		result := server.DB.DB.First(modelPlan, id)
		if result.Error == nil {
			c.JSON(fasthttp.StatusOK,
				webapi.GetPlan(*modelPlan),
			)
		} else {
			log.Error(result.Error)
			c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
		}
	} else {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusBadRequest, err)
	}
}

func (server *Server) GetPlan(c *gin.Context) {
	start, okStart := c.GetQuery("_start")
	end, okEnd := c.GetQuery("_end")
	sort := c.Query("_sort")
	order := c.Query("_order")
	var modelPlans []model.Plan
	var total int64
	var contentRange int64
	if okStart && okEnd {
		result := server.DB.DB.Where("id BETWEEN ? and ?", start, end).Order(fmt.Sprintf("%s %s", sort, order)).Find(&modelPlans).Count(&total)
		if result.Error == nil {
			server.DB.DB.Table("plans").Count(&contentRange)
			c.Writer.Header().Set("X-Total-Count", strconv.FormatInt(contentRange, 10))
			c.JSON(fasthttp.StatusOK, webapi.GetPlans(modelPlans))
		} else {
			c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
		}
	} else {
		id, okID := c.GetQuery("id")
		if okID {
			modelPlan := &model.Plan{}
			result := server.DB.DB.First(modelPlan, id)
			if result.Error == nil {
				c.Writer.Header().Set("X-Total-Count", "1")
				c.JSON(fasthttp.StatusOK,
					[]webapi.Plan{webapi.GetPlan(*modelPlan)},
				)
			} else {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			}
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) PostMeta(c *gin.Context) {

}

func (server *Server) DeleteMeta(c *gin.Context) {

}

func (server *Server) PutMeta(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		meta := &model.Meta{}
		server.DB.DB.Model(meta).First(&meta)
		request := &webapi.Meta{}
		err := c.BindJSON(request)
		if err == nil {
			meta.Title = request.Title
			meta.Desc = request.Desc
			server.DB.DB.Save(meta)
			c.JSON(fasthttp.StatusOK, request)
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) GetMeta(c *gin.Context) {
	meta := &model.Meta{}
	var count int64
	server.DB.DB.Model(meta).First(&meta).Count(&count)
	if count == 0 {
		meta.Title = "default"
		meta.Desc = "default"
		server.DB.DB.Save(meta)
	}
	c.JSON(fasthttp.StatusOK, webapi.Meta{
		Title: meta.Title,
		Desc:  meta.Desc,
	})
}

func (server *Server) PostNotice(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		request := &webapi.PostNoticeRequest{}
		err := c.BindJSON(request)
		if err == nil {
			notice := model.Notice{
				Text:  request.Text,
				Title: request.Title,
			}
			result := server.DB.DB.Save(&notice)
			if result.Error != nil {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			} else {
				c.JSON(fasthttp.StatusOK, webapi.GetNotice(notice))
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) DeleteNotice(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			result := server.DB.DB.Delete(&model.Notice{}, id)
			if result.Error == nil {
				c.JSON(fasthttp.StatusOK, id)
			} else {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) PutNotice(c *gin.Context) {
	role, err := server.GetRole(c)
	if err == nil && role == model.RoleAdmin {
		request := &webapi.Notice{}
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err == nil {
			err = c.BindJSON(request)
			if err == nil {
				result := server.DB.DB.Model(&model.Notice{
					Model: gorm.Model{
						ID: uint(id),
					},
				}).Updates(model.Notice{
					Model: gorm.Model{
						UpdatedAt: time.Now(),
					},
					Title: request.Title,
					Text:  request.Text,
				})
				if result.Error != nil {
					log.Error(result.Error)
					c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
				} else {
					c.JSON(fasthttp.StatusOK, request)
				}
			} else {
				log.Error(err)
				c.AbortWithError(fasthttp.StatusBadRequest, err)
			}
		} else {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		}
	} else {
		if err != nil {
			log.Error(err)
			c.AbortWithError(fasthttp.StatusBadRequest, err)
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) GetNoticeOne(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err == nil {
		modelNotice := &model.Notice{}
		result := server.DB.DB.First(modelNotice, id)
		if result.Error == nil {
			c.JSON(fasthttp.StatusOK,
				webapi.GetNotice(*modelNotice),
			)
		} else {
			log.Error(result.Error)
			c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
		}
	} else {
		log.Error(err)
		c.AbortWithError(fasthttp.StatusBadRequest, err)
	}
}

func (server *Server) GetNotice(c *gin.Context) {
	start, okStart := c.GetQuery("_start")
	end, okEnd := c.GetQuery("_end")
	sort := c.Query("_sort")
	order := c.Query("_order")
	var modelNotices []model.Notice
	var total int64
	var contentRange int64
	if okStart && okEnd {
		result := server.DB.DB.Where("id BETWEEN ? and ?", start, end).Order(fmt.Sprintf("%s %s", sort, order)).Find(&modelNotices).Count(&total)
		if result.Error == nil {
			server.DB.DB.Table("notices").Count(&contentRange)
			c.Writer.Header().Set("X-Total-Count", strconv.FormatInt(contentRange, 10))
			c.JSON(fasthttp.StatusOK, webapi.GetNotices(modelNotices))
		} else {
			c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
		}
	} else {
		id, okID := c.GetQuery("id")
		if okID {
			modelNotice := &model.Notice{}
			result := server.DB.DB.First(modelNotice, id)
			if result.Error == nil {
				c.Writer.Header().Set("X-Total-Count", "1")
				c.JSON(fasthttp.StatusOK,
					[]webapi.Notice{webapi.GetNotice(*modelNotice)},
				)
			} else {
				log.Error(result.Error)
				c.AbortWithError(fasthttp.StatusBadRequest, result.Error)
			}
		} else {
			c.AbortWithStatus(fasthttp.StatusBadRequest)
		}
	}
}

func (server *Server) GetRole(c *gin.Context) (uint, error) {
	session := sessions.Default(c)
	userIDStr := session.Get("userID")
	if userIDStr != nil {
		if userIDStr == "admin" {
			return model.RoleAdmin, nil
		} else {
			userID, ok := userIDStr.(uint)
			if ok {
				user := &model.User{}
				result := server.DB.DB.First(user, userID)
				if result.Error == nil {
					return user.Role, nil
				} else {
					return model.RoleNone, result.Error
				}
			} else {
				return model.RoleNone, nil
			}
		}
	} else {
		return model.RoleNone, nil
	}
}
