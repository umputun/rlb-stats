<!DOCTYPE html>
<html>
<head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta http-equiv="refresh" content="300"> <!-- Refresh every 5 minutes -->
    <title>rlb-stats</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css" integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">
</head>
<body>
<table class="table">
    <thead>
    <tr>
        <th scope="col" style="width: 50%">Top Files</th>
        <th scope="col" style="width: 50%">Top Nodes</th>
    </tr>
    </thead>
    <tbody>
    <tr>
        <td><img src="https://raw.githubusercontent.com/zieckey/gochart/master/image/spline.png" class="img-fluid"></td>
        <td><img src="https://raw.githubusercontent.com/zieckey/gochart/master/image/spline.png" class="img-fluid"></td>
    </tr>
    <tr>
        <td>
            <table class="table table-hover">
                <thead>
                <tr>
                    <th scope="col">File Name</th>
                    <th scope="col">Volume</th>
                </tr>
                </thead>
                <tbody>
                {{ range .TopFiles }}
                <tr>
                    <th scope="row"><a href="/file_stats?name={{.Name}}">{{.Name}}</a></th>
                    <td>{{.Volume}}</td>
                </tr>
                {{ end }}
                </tbody>
            </table>
        </td>
        <td>
            <table class="table table-hover">
                <thead>
                <tr>
                    <th scope="col">Node Name</th>
                    <th scope="col">Volume</th>
                </tr>
                </thead>
                <tbody>
                {{ range .TopNodes }}
                <tr>
                    <th scope="row">{{.Name}}</th>
                    <td>{{.Volume}}</td>
                </tr>
                {{ end }}
                </tbody>
            </table>
        </td>
    </tr>
    </tbody>
</table>
</body>
</html>