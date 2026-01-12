package http

// createController регистрирует публичные (mobile/Web) эндпоинты.
// Префикс /app сохранён для обратной совместимости.
func (s *Server) createController() {
	common := s.serv.Group("api")

	userGroup := common.Group("user")
	userGroup.POST("/register", s.Register)
	userGroup.POST("/login", s.Login)

	AuthUserGroup := userGroup.Use(s.withLogger).Use(s.auth)
	AuthUserGroup.POST("/orders", s.UploadOrder)
	AuthUserGroup.GET("/orders", s.GetOrders)

	//AuthUserGroup.GET("/balance", s.Ping)
	AuthUserGroup.POST("/balance/withdraw", s.UploadWithdraw)
	AuthUserGroup.GET("/withdrawals", s.GetWithdraw)
}
