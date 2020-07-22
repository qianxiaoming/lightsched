#include "httputil.h"
#include <boost/format.hpp>

namespace lightsched {

HttpClient::HttpClient()
{
}

HttpClient::~HttpClient()
{
    if (stream) {
        beast::error_code ec;
        stream->socket().shutdown(tcp::socket::shutdown_both, ec);
    }
}

bool HttpClient::Connect(const std::string& server, uint16_t port)
{
    try {
        tcp::resolver resolver(ioc);
        auto const results = resolver.resolve(server, boost::str(boost::format("%d") % port));
        stream.reset(new beast::tcp_stream(ioc));
        stream->connect(results);
        server_host = server;
    }
    catch (std::exception const& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        stream.reset();
        return false;
    }
    return true;
}

http::status HttpClient::Get(const std::string& target, std::string& response)
{
    http::request<http::string_body> req{ http::verb::get, target, 11 };
    req.set(http::field::host, server_host);
    req.set(http::field::user_agent, BOOST_BEAST_VERSION_STRING);
    http::write(*stream, req);

    beast::flat_buffer buffer;
    http::response<http::string_body> res;
    http::read(*stream, buffer, res);
    response = res.body();
    return res.result();
}

http::status HttpClient::Post(const std::string& target, const std::string& body, std::string& response)
{
    http::request<http::string_body> req{ http::verb::post, target, 11 };
    req.set(http::field::host, server_host);
    req.set(http::field::user_agent, BOOST_BEAST_VERSION_STRING);
    if (!body.empty()) {
        req.body() = body;
        req.prepare_payload();
    }
    http::write(*stream, req);

    beast::flat_buffer buffer;
    http::response<http::string_body> res;
    http::read(*stream, buffer, res);
    response = res.body();
    return res.result();
}

http::status HttpClient::Put(const std::string& target, const std::string& body, std::string& response)
{
    http::request<http::string_body> req{ http::verb::put, target, 11 };
    req.set(http::field::host, server_host);
    req.set(http::field::user_agent, BOOST_BEAST_VERSION_STRING);
    if (!body.empty()) {
        req.body() = body;
        req.prepare_payload();
    }
    http::write(*stream, req);

    beast::flat_buffer buffer;
    http::response<http::string_body> res;
    http::read(*stream, buffer, res);
    response = res.body();
    return res.result();
}

http::status HttpClient::Delete(const std::string& target, std::string& response)
{
    http::request<http::string_body> req{ http::verb::delete_, target, 11 };
    req.set(http::field::host, server_host);
    req.set(http::field::user_agent, BOOST_BEAST_VERSION_STRING);
    http::write(*stream, req);

    beast::flat_buffer buffer;
    http::response<http::string_body> res;
    http::read(*stream, buffer, res);
    response = res.body();
    return res.result();
}

}
