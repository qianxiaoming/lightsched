#include <cstdlib>
#include <iostream>
#include <string>
#include <boost/beast.hpp>
#include <boost/asio/connect.hpp>
#include <boost/asio/ip/tcp.hpp>

namespace beast = boost::beast;     // from <boost/beast.hpp>
namespace http = beast::http;       // from <boost/beast/http.hpp>
namespace net = boost::asio;        // from <boost/asio.hpp>
using tcp = net::ip::tcp;           // from <boost/asio/ip/tcp.hpp>

namespace lightsched {

class HttpClient
{
public:
	HttpClient();

	~HttpClient();

	bool Connect(const std::string& server, uint16_t port);

	bool IsConnected() const { return stream.get() != nullptr; }

	http::status Get(const std::string& target, std::string& response);

private:
	std::string server_host;
	boost::asio::io_context ioc;
	boost::shared_ptr<boost::beast::tcp_stream> stream;
};

}