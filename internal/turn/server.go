package turn

// Функция запуска TURN-сервера с TLS и UDP
//func startTurnServer(turnCfg *config.TurnServerConfig) (*turn.Server, error) {
//	// TLS для TCP listener
//	//cert, err := tls.LoadX509KeyPair(turnCfg.CertFile, turnCfg.KeyFile)
//	//if err != nil {
//	//	return nil, fmt.Errorf("load cert: %w", err)
//	//}
//
//	tcpListener, err := net.Listen(
//		"tcp",
//		fmt.Sprintf(":%d", turnCfg.Port),
//		//&tls.Config{
//		//	Certificates: []tls.Certificate{cert},
//		//},
//	)
//	if err != nil {
//		return nil, fmt.Errorf("tcp listen: %w", err)
//	}
//
//	udpListener, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", turnCfg.Port))
//	if err != nil {
//		return nil, fmt.Errorf("udp listen: %w", err)
//	}
//
//	relayAddressGenerator := &turn.RelayAddressGeneratorStatic{
//		RelayAddress: net.ParseIP(turnCfg.PublicIP),
//		Address:      "0.0.0.0",
//	}
//
//	server, err := turn.NewServer(
//		turn.ServerConfig{
//			Realm: turnCfg.Realm,
//			AuthHandler: func(username, realm string, srcAddr net.Addr) ([]byte, bool) {
//				return turn.GenerateAuthKey(turnCfg.Username, realm, turnCfg.Password), username == turnCfg.Username
//			},
//			ListenerConfigs: []turn.ListenerConfig{
//				{
//					Listener:              tcpListener,
//					RelayAddressGenerator: relayAddressGenerator,
//				},
//			},
//			PacketConnConfigs: []turn.PacketConnConfig{
//				{
//					PacketConn:            udpListener,
//					RelayAddressGenerator: relayAddressGenerator,
//				},
//			},
//		})
//	if err != nil {
//		return nil, fmt.Errorf("new turn server: %w", err)
//	}
//
//	slog.Info(
//		fmt.Sprintf(
//			"TURN server started on %s:%d (TCP+TLS, UDP)",
//			turnCfg.Host,
//			turnCfg.Port,
//		),
//	)
//
//	return server, nil
//}

// Handler для выдачи TURN-кредитов
//func (h *HttpHandler) turnCredentialsHandler(c echo.Context) error {
//	expiration := time.Now().Add(5 * time.Minute).Unix()
//	username := fmt.Sprintf("%d", expiration)
//	password := turn.GenerateAuthKey(username, h.cfg.TurnServer.Realm, h.cfg.TurnServer.Password)
//
//	resp := map[string]interface{}{
//		"username": username,
//		"password": password,
//		"ttl":      300, // 5 минут
//		"urls": []string{
//			fmt.Sprintf(
//				"turn:%s:%d?transport=udp",
//				h.cfg.TurnServer.Host,
//				h.cfg.TurnServer.Port,
//			),
//			fmt.Sprintf(
//				"turn:%s:%d?transport=tcp",
//				h.cfg.TurnServer.Host,
//				h.cfg.TurnServer.Port,
//			),
//		},
//	}
//
//	return c.JSON(http.StatusOK, resp)
//}
