<!DOCTYPE html>
<html>
<head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta http-equiv="refresh" content="300"> <!-- Refresh every 5 minutes -->
    <title>rlb-stats: {{.Filename}}</title>
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.2.1/css/bootstrap.min.css" integrity="sha384-GJzZqFGwb1QTTN6wy59ffF1BuGJpLSa9DkKMp0DgiMDm4iYMj70gZWKYbI706tWS" crossorigin="anonymous">
</head>
<body>
<table class="table">
    <thead>
    <tr>
        <th scope="col" style="width: 100%">{{.Filename}} stats</th>
    </tr>
    </thead>
    <tbody>
    <tr>
        <td><img src="/chart?type=by_file&filename={{.Filename}}{{if .From}}&from={{.From}}{{end}}{{if .To}}&to={{.To}}{{end}}" class="img-fluid" alt="{{$.Filename}} download stats"></td>
    </tr>
    </tbody>
</table>
</body>
</html>