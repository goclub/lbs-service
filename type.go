package main

type TLBSIPReply struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Result    struct {
		IP       string `json:"ip"`
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		AdInfo struct {
			Nation     string `json:"nation"`
			Province   string `json:"province"`
			City       string `json:"city"`
			District   string `json:"district"`
			Adcode     int    `json:"adcode"`
			NationCode int    `json:"nation_code"`
		} `json:"ad_info"`
	} `json:"result"`
}
type TLBSGeoReply struct {
	Status    int    `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Result    struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		Address          string `json:"address"`
		AddressComponent struct {
			Nation       string `json:"nation"`
			Province     string `json:"province"`
			City         string `json:"city"`
			District     string `json:"district"`
			Street       string `json:"street"`
			StreetNumber string `json:"street_number"`
		} `json:"address_component"`
		AdInfo struct {
			NationCode    string `json:"nation_code"`
			Adcode        string `json:"adcode"`
			PhoneAreaCode string `json:"phone_area_code"`
			CityCode      string `json:"city_code"`
			Name          string `json:"name"`
			Location      struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			Nation   string `json:"nation"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Distance int    `json:"_distance"`
		} `json:"ad_info"`
		AddressReference struct {
			Town struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance int    `json:"_distance"`
				DirDesc  string `json:"_dir_desc"`
			} `json:"town"`
			LandmarkL2 struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"landmark_l2"`
			Street struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"street"`
			StreetNumber struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"street_number"`
			Crossroad struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
				Distance float64 `json:"_distance"`
				DirDesc  string  `json:"_dir_desc"`
			} `json:"crossroad"`
		} `json:"address_reference"`
		FormattedAddresses struct {
			Recommend       string `json:"recommend"`
			Rough           string `json:"rough"`
			StandardAddress string `json:"standard_address"`
		} `json:"formatted_addresses"`
	} `json:"result"`
}
