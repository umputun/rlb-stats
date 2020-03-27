const chartInstances = new WeakMap();

function convertDataToTauChartFormat(data) {
  const chartData = [];
  data.forEach(({file, node, data: dataSeries}) => {
    dataSeries.forEach(([x, y]) => {
      const date = new Date(x);
      const volume = y;
      if (file) {
        chartData.push({
          file,
          date,
          volume
        })
      } else {
        chartData.push({
          node,
          date,
          volume
        })
      }
    });
  });
  return chartData;
}

export async function drawChart({data, container}) {
  container.className = `chart tauchart`;
  const chartData = convertDataToTauChartFormat(data);
  const lineName = chartData[0].file ? "file" : "node";
  const config = {
    data: chartData,
    type: "line",
    x: "date",
    y: "volume",
    color: lineName,
    guide: {
      x: {nice: false}
    },
    plugins: [
      Taucharts.api.plugins.get("legend")({
      //   position: "bottom"
      }),
      Taucharts.api.plugins.get("tooltip")()
    ]
  };
  if (chartInstances.has(container)) {
    chartInstances.get(container).updateConfig(config);
    return;
  }

  const chart = new Taucharts.Chart(config);
  chart.renderTo(container);
  chartInstances.set(container, chart);
}
