def Echo(request, response):
    response.message = "Hello " + request.name + "!"
    return response
