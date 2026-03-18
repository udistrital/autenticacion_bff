package controllers

import (
	beego "github.com/beego/beego/v2/server/web"
)

// MeController operations for Me
type MeController struct {
	beego.Controller
}

// URLMapping ...
func (c *MeController) URLMapping() {
	c.Mapping("Post", c.Post)
	c.Mapping("GetOne", c.GetOne)
	c.Mapping("GetAll", c.GetAll)
	c.Mapping("Put", c.Put)
	c.Mapping("Delete", c.Delete)
}

// Post ...
// @Title Create
// @Description create Me
// @Param	body		body 	models.Me	true		"body for Me content"
// @Success 201 {object} models.Me
// @Failure 403 body is empty
// @router / [post]
func (c *MeController) Post() {

}

// GetOne ...
// @Title GetOne
// @Description get Me by id
// @Param	id		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.Me
// @Failure 403 :id is empty
// @router /:id [get]
func (c *MeController) GetOne() {

}

// GetAll ...
// @Title GetAll
// @Description get Me
// @Param	query	query	string	false	"Filter. e.g. col1:v1,col2:v2 ..."
// @Param	fields	query	string	false	"Fields returned. e.g. col1,col2 ..."
// @Param	sortby	query	string	false	"Sorted-by fields. e.g. col1,col2 ..."
// @Param	order	query	string	false	"Order corresponding to each sortby field, if single value, apply to all sortby fields. e.g. desc,asc ..."
// @Param	limit	query	string	false	"Limit the size of result set. Must be an integer"
// @Param	offset	query	string	false	"Start position of result set. Must be an integer"
// @Success 200 {object} models.Me
// @Failure 403
// @router / [get]
func (c *MeController) GetAll() {

}

// Put ...
// @Title Put
// @Description update the Me
// @Param	id		path 	string	true		"The id you want to update"
// @Param	body		body 	models.Me	true		"body for Me content"
// @Success 200 {object} models.Me
// @Failure 403 :id is not int
// @router /:id [put]
func (c *MeController) Put() {

}

// Delete ...
// @Title Delete
// @Description delete the Me
// @Param	id		path 	string	true		"The id you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 id is empty
// @router /:id [delete]
func (c *MeController) Delete() {

}
