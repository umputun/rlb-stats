async function loadData(args) {
  const url = `http://127.0.0.1:8080/api/candle?from=${args.from.toISOString()}&to=${args.to.toISOString()}&aggregate=${
    args.aggregate
  }m`;
  try {
    const response = await fetch(url);
    const jsonData = await response.json();
    return jsonData;
  } catch (error) {
    console.log(error);
  }
}

function transformToFiles(jsonData) {
  const filesData = jsonData.reduce(
    (acc, element) => {
      acc.files[0].data.push([
        new Date(element.StartMinute).getTime(),
        element.Nodes.all.Volume
      ]);
      Object.keys(element.Nodes.all.Files).forEach(file => {
        if (!acc.files.find(el => el.file === file)) {
          acc.files.push({
            file: file,
            data: [
              [
                new Date(element.StartMinute).getTime(),
                element.Nodes.all.Files[file]
              ]
            ]
          });
        } else {
          acc.files.forEach(el => {
            if (el.file === file) {
              el.data.push([
                new Date(element.StartMinute).getTime(),
                element.Nodes.all.Files[file]
              ]);
            }
          });
        }
      });
      Object.keys(element.Nodes).forEach(node => {
        if (!acc.nodes.find(el => el.node === node)) {
          acc.nodes.push({
            node: node,
            data: [
              [
                new Date(element.StartMinute).getTime(),
                element.Nodes[node].Volume
              ]
            ]
          });
        } else {
          acc.nodes.forEach(el => {
            if (el.node === node) {
              el.data.push([
                new Date(element.StartMinute).getTime(),
                element.Nodes[node].Volume
              ]);
            }
          });
        }
      });

      return acc;
    },
    { files: [{ file: "all", data: [] }], nodes: [] }
  );
  const sortedFilesData = filesData.files.sort((a, b) => {
    return (
      b.data.reduce((acc, element) => {
        return (acc += element[1]);
      }, 0) -
      a.data.reduce((acc, element) => {
        return (acc += element[1]);
      }, 0)
    );
  });

  // choose only top 10 files or nodes and total qty
  const sortedNodesData = filesData.nodes.sort((a, b) => {
    return (
      b.data.reduce((acc, element) => {
        return (acc += element[1]);
      }, 0) -
      a.data.reduce((acc, element) => {
        return (acc += element[1]);
      }, 0)
    );
  });
  return {
    files: sortedFilesData.slice(0, 11),
    nodes: sortedNodesData.slice(0, 11)
  };
}

// date and time format for title and tooltips
const dateTimeOptions = {
  year: "numeric",
  month: "short",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit"
};

// default charts are for last 10 days with 144 minutes aggregate
async function loadAndDraw({
  from = new Date(new Date().setDate(new Date().getDate() - 10)),
  to = new Date(),
  aggregate = 144
} = {}) {
  const jsonData = await loadData({ to, from, aggregate });
  const data = transformToFiles(jsonData);
  drawChart(
    data.files,
    "top-files",
    `Top 10 files downloaded from ${from.toLocaleDateString(
      [],
      dateTimeOptions
    )}`
  );
  drawChart(
    data.nodes,
    "top-nodes",
    `Top 10 nodes from ${from.toLocaleDateString([], dateTimeOptions)}`
  );
}

anychart.onDocumentLoad(() => {
  loadAndDraw();
});

const buttons = document.getElementsByClassName("period-button");
for (let button of buttons) {
  button.onclick = event => {
    let args = {};
    if (event.target.id === "1hour") {
      args = {
        from: new Date(new Date().setHours(new Date().getHours() - 1)),
        aggregate: 1
      };
    } else if (event.target.id === "12hours") {
      args = {
        from: new Date(new Date().setHours(new Date().getHours() - 12)),
        aggregate: 1
      };
    } else if (event.target.id === "24hours") {
      args = {
        from: new Date(new Date().setDate(new Date().getDate() - 1)),
        aggregate: 1
      };
    } else if (event.target.id === "10days") {
      args = {
        from: new Date(new Date().setDate(new Date().getDate() - 10)),
        aggregate: 144
      };
    }
    loadAndDraw(args);
  };
}

function drawChart(data, container, title) {
  //clear the container
  document.getElementById(container).innerHTML = `<h4>${title}</h4>`;
  const chart = anychart.line();
  const lines = data.map(el => {
    const line = chart.spline(el.data);
    if (el.file) {
      line.name(el.file);
    } else if (el.node) {
      line.name(el.node);
    }
    return line;
  });
  // set the container element
  chart.container(container);

  // enable the legend
  chart.legend(true);
  // set the layout of the legend
  chart.legend().itemsLayout("vertical-expandable");
  // set the position of the legend
  chart.legend().position("bottom");
  // set the maximum height and width of the legend
  chart.legend().height("220px");
  chart.height("500px");
  const margin = chart.legend().margin();
  margin.top(10);

  // enable html for the legend
  chart.legend().useHtml(true);

  // configure the format of legend items
  chart
    .legend()
    .itemsFormat(
      "<span style='color:#455a64;font-weight:600'>{%seriesName}:</span> {%seriesYSum}"
    );

  // configure the legend paginator
  const paginator = chart.legend().paginator();
  paginator.layout("vertical");
  paginator.fontSize(9);

  const labels = chart.xAxis().labels();
  labels.hAlign("center");
  labels.width(60);
  labels.fontSize(10);
  const xAxisLabels = chart.xAxis().labels();
  xAxisLabels.rotation(90);

  // create custom Date Time scale
  const dateTimeScale = anychart.scales.dateTime();
  const dateTimeTicks = dateTimeScale.ticks();

  // get period of time from the first and the last point in hours
  const periodInHours =
    (data[0].data[data[0].data.length - 1][0] - data[0].data[0][0]) / 3600000;

  // format the x axis ticks
  if (periodInHours <= 1) {
    dateTimeTicks.interval("minutes", 5);
  } else if (periodInHours <= 6) {
    dateTimeTicks.interval("minutes", 20);
  } else if (periodInHours <= 12) {
    dateTimeTicks.interval("hours", 1);
  } else if (periodInHours <= 24) {
    dateTimeTicks.interval("hours", 2);
  } else if (periodInHours <= 240) {
    dateTimeTicks.interval("days", 1);
  } else if (periodInHours <= 720) {
    dateTimeTicks.interval("days", 3);
  }

  // apply Date Time scale
  chart.xScale(dateTimeScale);

  // set the alignment of the legend
  chart.legend().align("left");

  // tooltip settings
  const tooltip = chart.tooltip();
  tooltip.titleFormat(data => {
    return new Date(parseInt(data.kc.values.x.value)).toLocaleDateString(
      [],
      dateTimeOptions
    );
  });

  // initiate chart display
  chart.draw();
}
