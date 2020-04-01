let charts = {};

export function drawChart({ data, container, title }) {
  container.className = `chart apexchart`;
  const dataType = data.length > 0 && data[0].file ? "file" : "node";
  const series = data.map(datum => {
    return { name: datum[dataType], data: datum.data };
  });
  var options = {
    chart: {
      type: "line",
      height: 700
    },
    stroke: {
      curve: "smooth",
      width: 3
    },
    series,
    xaxis: {
      type: "datetime"
    },
    title: {
      text: title,
      align: "center",
      offsetY: 15,
      style: {
        fontSize: "16px",
        fontWeight: "bold",
        color: "rgb(69, 90, 101)"
      }
    },
    legend: {
      itemMargin: {
        horizontal: 5,
        vertical: 5
      },
      offsetY: 10,
      height: 150,
      formatter: function(seriesName, opts) {
        return [
          `${seriesName}:`,
          opts.w.globals.series[opts.seriesIndex].reduce(
            (sum, datum) => sum + datum,
            0
          )
        ];
      }
    },
    responsive: [
      {
        breakpoint: 800,
        options: {
          legend: {
            fontSize: "11px",
            height: 100,
            offsetY: 5,
            itemMargin: {
              horizontal: 5,
              vertical: 0
            }
          },
          title: {
            style: { fontSize: "12px" }
          },
          chart: {
            height: 400
          }
        }
      }
    ]
  };
  if (!charts[dataType]) {
    charts[dataType] = new ApexCharts(container, options);
    charts[dataType].render();
  } else {
    charts[dataType].updateOptions(options);
  }
}
