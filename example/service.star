def Echo(request, response):
    response.message = "Hello " + request.name + "!"
    response.timestamp = now()
    return response
