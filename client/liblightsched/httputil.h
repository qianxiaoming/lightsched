#include <cstdlib>
#include <iostream>
#include <string>
#include <boost/beast.hpp>
#include <boost/asio/connect.hpp>
#include <boost/asio/ip/tcp.hpp>

namespace lightsched {

class HttpClient
{
public:
	HttpClient();

	~HttpClient();

	bool Connect(const std::string& server, uint16_t port);

	int Get(const std::string& target, std::string& response);

private:
	std::string server_host;
	boost::asio::io_context ioc;
	boost::beast::tcp_stream* stream;
};

}