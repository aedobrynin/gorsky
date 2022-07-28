
# Prokudin-Gorsky's photo processing

The program makes colored photo from S.M. Prokudin-Gorsky's negatives. 

![Algo result #1](https://github.com/hashlib/Prokudin-Gorsky/blob/master/results_for_readme/2018678905.png)

![Algo result #2](https://github.com/hashlib/Prokudin-Gorsky/blob/master/results_for_readme/2018679120.png)

![Algo result #3](https://github.com/hashlib/Prokudin-Gorsky/blob/master/results_for_readme/2018679802.png)

## Install

```
go install github.com/aedobrynin/gorsky@latest
```

Run for *your_image.tif* (see ```gorsky --help``` for more information):
```
gorsky your_image.tif
```
The result will be saved in the *result* directory.

## Algorithm explanation
The algorithms finds the best shifts for image channels using [correlation coefficient](https://en.wikipedia.org/wiki/Correlation_coefficient). This process is sped up by [image pyramid](https://en.wikipedia.org/wiki/Pyramid_(image_processing)) and wide use of goroutines.

## Algorithm results
 You can find algorithm results [here](https://bit.ly/2EIYNYq).

## How to get negatives
 Visit [the collection website](https://www.loc.gov/collections/prokudin-gorskii/) to download negatives. You need an image with description `[ digital file from glass neg. ]`. The image should look like this:
 <p align="center">
  <img src="https://tile.loc.gov/storage-services/service/pnp/prok/00900/00924v.jpg" />
 </p>
