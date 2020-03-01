loadAndDraw();

async function loadData(args) {
  const url = `/api/candle?from=${args.from.toISOString()}&to=${args.to.toISOString()}`;
  const errorParagraph = document.getElementById("error");
  errorParagraph.innerHTML = "";
  try {
    const response = await fetch(url);
    if (!response.ok) {
      errorParagraph.innerHTML = `Error retrieving data for graphs: ${response.status} ${response.statusText}`;
      return [];
    }
    return await response.json();
  } catch (error) {
    errorParagraph.innerHTML = `Error retrieving data for graphs: ${error.message}`;
    return [];
  }
}

function transformToFiles(jsonData) {
  if (jsonData.length === 0) {
    return {files: [], nodes: []};
  }
  const filesData = jsonData.reduce(
    (acc, element) => {
      const entryDate = new Date(element.StartMinute).getTime();

      // all files volume to given minute
      acc.files[0].data.push([entryDate, element.Nodes.all.Volume]);

      // per-file stats for given minute
      Object.keys(element.Nodes.all.Files).forEach(file => {
        if (!acc.files.find(el => el.file === file)) {
          acc.files.push({
            file: file,
            data: [[entryDate, element.Nodes.all.Files[file]]]
          });
        } else {
          acc.files.forEach(el => {
            if (el.file === file) {
              el.data.push([entryDate, element.Nodes.all.Files[file]]);
            }
          });
        }
      });

      // per-node stats for given minute
      Object.keys(element.Nodes).forEach(node => {
        if (!acc.nodes.find(el => el.node === node)) {
          acc.nodes.push({
            node: node,
            data: [[entryDate, element.Nodes[node].Volume]]
          });
        } else {
          acc.nodes.forEach(el => {
            if (el.node === node) {
              el.data.push([entryDate, element.Nodes[node].Volume]);
            }
          });
        }
      });

      return acc;
    },
    {files: [{file: "all", data: []}], nodes: []}
  );

  const sortedFilesData = sortFilesOrNodes(filesData.files);
  const sortedNodesData = sortFilesOrNodes(filesData.nodes);
  // choose only top 10 files or nodes and total qty
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

// default charts are for last 24 hours
async function loadAndDraw(minutes = 24 * 60) {
  document.title = `rlb-stats: data for the last ${getReadableDuration(
    minutes
  )}`;
  const args = {
    from: new Date(new Date().setMinutes(new Date().getMinutes() - minutes)),
    to: new Date()
  };
  const jsonData = await loadData(args);
  const data = transformToFiles(jsonData);
  drawChart(
    data.files,
    "top-files",
    `Top 10 files downloaded from ${args.from.toLocaleDateString(
      [],
      dateTimeOptions
    )}`
  );
  drawChart(
    data.nodes,
    "top-nodes",
    `Top 10 nodes from ${args.from.toLocaleDateString([], dateTimeOptions)}`
  );
}

const buttons = document.getElementById("period-buttons");
buttons.addEventListener("click", event =>
  loadAndDraw(parseInt(event.target.dataset.minutes))
);

function drawChart(data, container, title) {
  //clear the container
  document.getElementById(container).innerHTML = `<h4>${title}</h4>`;
  const chart = anychart.line();
  data.map(elem => {
    const line = chart.spline(elem.data);
    if (elem.file) {
      line.name(elem.file);
    } else if (elem.node) {
      line.name(elem.node);
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
  let periodInHours = 1;
  if (data.length > 0) {
    periodInHours =
      (data[0].data[data[0].data.length - 1][0] - data[0].data[0][0]) /
      (1000 * 60 * 60);
  }
  // format the x axis ticks
  switch (true) {
    case periodInHours <= 1:
      dateTimeTicks.interval("minutes", 5);
      break;
    case periodInHours <= 6:
      dateTimeTicks.interval("minutes", 20);
      break;
    case periodInHours <= 12:
      dateTimeTicks.interval("hours", 1);
      break;
    case periodInHours <= 24:
      dateTimeTicks.interval("hours", 2);
      break;
    case periodInHours <= 240:
      dateTimeTicks.interval("days", 1);
      break;
    case periodInHours <= 720:
      dateTimeTicks.interval("days", 3);
      break;
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

function getReadableDuration(minutes) {
  let tempMinutes = minutes;
  let readableDate = "";
  if (tempMinutes >= 60 * 24) {
    readableDate += String(Math.floor(tempMinutes / (60 * 24))) + "d";
    tempMinutes = tempMinutes % (60 * 24);
  }
  if (tempMinutes >= 60) {
    readableDate += String(Math.floor(tempMinutes / 60)) + "h";
    tempMinutes = tempMinutes % 60;
  }
  if (tempMinutes > 0) {
    readableDate += `${tempMinutes}m`;
  }
  return readableDate;
}

function sortFilesOrNodes(arr) {
  return arr.sort((a, b) => {
    return b.data.reduce((acc, element) => {
      return acc + element[1];
    }, 0) - a.data.reduce((acc, element) => {
      return acc + element[1];
    }, 0);
  });
}
