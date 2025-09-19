package turn

// TODO: на будущее для своего turn сервера
//type TurnServerConfig struct {
//	PublicIP string `env:"PUBLIC_IP" envDefault:"0.0.0.0"`
//	Host     string `env:"TURN_HOST,required"`
//	Port     int    `env:"TURN_PORT" envDefault:"3478"`
//	Realm    string `env:"TURN_REALM" envDefault:"xxsm.ru"`
//	Username string `env:"TURN_USERNAME,required"`
//	Password string `env:"TURN_PASSWORD,required"`
//	CertFile string `env:"TURN_CERT_FILE" envDefault:"/etc/certs/tls.crt"`
//	KeyFile  string `env:"TURN_KEY_FILE" envDefault:"/etc/certs/tls.key"`
//}
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
