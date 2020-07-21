#include "httputil.h"
#include <boost/format.hpp>

namespace beast = boost::beast;     // from <boost/beast.hpp>
namespace http = beast::http;       // from <boost/beast/http.hpp>
namespace net = boost::asio;        // from <boost/asio.hpp>
using tcp = net::ip::tcp;           // from <boost/asio/ip/tcp.hpp>

namespace lightsched {

HttpClient::HttpClient() : stream(nullptr)
{
}

HttpClient::~HttpClient()
{
    delete stream;
    stream = nullptr;
}

bool HttpClient::Connect(const std::string& server, uint16_t port)
{
    try {
        tcp::resolver resolver(ioc);
        auto const results = resolver.resolve(server, boost::str(boost::format("%d") % port));
        stream = new beast::tcp_stream(ioc);
        stream->connect(results);
        server_host = server;
    }catch(std::exception const& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        delete stream;
        stream = nullptr;
        return false;
    }
    return true;
}

int HttpClient::Get(const std::string& target, std::string& response)
{
    try {
        http::request<http::string_body> req{ http::verb::get, target, 11 };
        req.set(http::field::host, server_host);
        req.set(http::field::user_agent, BOOST_BEAST_VERSION_STRING);
        http::write(*stream, req);

        beast::flat_buffer buffer;
        http::response<http::dynamic_body> res;
        http::read(stream, buffer, res);

        std::cout << res << std::endl;
        return res.result_int();
    } catch (std::exception const& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 0;
    }
}

}
