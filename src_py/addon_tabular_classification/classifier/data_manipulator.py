import pandas as pd
from scipy.io import arff
import matplotlib.pyplot as plt

def read_arff(f):
    data = arff.loadarff(f)
    return pd.DataFrame(data[0])

def remap_labels(cs, df, em):
    df[cs] = df[cs].astype('category')
    df[cs].replace(em, inplace=True)
    return df

if __name__ == '__main__':
    df = read_arff('../../artefacts/electricity-normalized.arff')
    print(df.head())
    encode_map = {
        b'UP': 1,
        b'DOWN': 0
    }
    df = remap_labels('class', df, encode_map)
    print(df.head())
