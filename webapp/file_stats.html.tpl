<!DOCTYPE html>
<html>
<head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta http-equiv="refresh" content="300"> <!-- Refresh every 5 minutes -->
    <title>rlb-stats: {{.Name}}</title>
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.2.1/css/bootstrap.min.css" integrity="sha384-GJzZqFGwb1QTTN6wy59ffF1BuGJpLSa9DkKMp0DgiMDm4iYMj70gZWKYbI706tWS" crossorigin="anonymous">
</head>
<body>
<table class="table">
    <thead>
    <tr>
        <th scope="col" style="width: 100%">{{.Name}} stats</th>
    </tr>
    </thead>
    <tbody>
    <tr>
        {{ range $_, $url := .Charts }}
        <td><img src="{{$url}}" class="img-fluid"></td>
        {{ end }}
    </tr>
    </tbody>
</table>
</body>
</html>