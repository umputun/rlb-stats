loadAndDrawE();

async function loadDataE(args) {
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

function transformToFilesE(jsonData) {
  if (jsonData.length === 0) {
    return { files: [], nodes: [] };
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
    { files: [{ file: "all", data: [] }], nodes: [] }
  );

  const sortedFilesData = sortFilesOrNodesE(filesData.files);
  const sortedNodesData = sortFilesOrNodesE(filesData.nodes);
  // choose only top 10 files or nodes and total qty
  return {
    files: sortedFilesData.slice(0, 11),
    nodes: sortedNodesData.slice(0, 11)
  };
}

// date and time format for title and tooltips
const dateTimeOptionsE = {
  year: "numeric",
  month: "short",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit"
};

// default charts are for last 24 hours
async function loadAndDrawE(minutes = 24 * 60) {
  document.title = `rlb-stats: data for the last ${getReadableDurationE(
    minutes
  )}`;
  const args = {
    from: new Date(new Date().setMinutes(new Date().getMinutes() - minutes)),
    to: new Date()
  };
  const jsonData = await loadDataE(args);
  const data = transformToFilesE(jsonData);
  drawChartE(
    data.files,
    "top-files-e",
    `Top 10 files downloaded from ${args.from.toLocaleDateString(
      [],
      dateTimeOptions
    )}`
  );
  drawChartE(
    data.nodes,
    "top-nodes-e",
    `Top 10 nodes from ${args.from.toLocaleDateString([], dateTimeOptions)}`
  );
}

const buttonsE = document.getElementById("period-buttons");
buttonsE.addEventListener(
  "click",
  event =>
    event.target.tagName === "BUTTON" &&
    loadAndDrawE(parseInt(event.target.dataset.minutes))
);

function drawChartE(data, container, title) {
  const myChart = echarts.init(document.getElementById(container));
  // specify chart configuration item and data
  var option = {
    title: {
      text: title
    },
    tooltip: {},
    legend: {
      data: ["Sales"]
    },
    xAxis: { type: "time" },
    yAxis: {},
    series: data.map(fileData => {
      return {
        name: fileData.file,
        encode: {
          x: 0,
          y: 1
        },
        type: "line",
        smooth: true,
        data: fileData.data.map(el => [el[0], el[1]])
      };
    })
  };

  // use configuration item and data specified to show chart
  myChart.setOption(option);
}

function getReadableDurationE(minutes) {
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

function sortFilesOrNodesE(arr) {
  return arr.sort((a, b) => {
    return (
      b.data.reduce((acc, element) => {
        return acc + element[1];
      }, 0) -
      a.data.reduce((acc, element) => {
        return acc + element[1];
      }, 0)
    );
  });
}
